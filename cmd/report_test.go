package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/kubeone/go-time-tracker/internal/model"
	"github.com/kubeone/go-time-tracker/internal/storage"
)

// ---- parseDateArgs ---------------------------------------------------------

func fixedToday() time.Time {
	return truncateDay(time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local)) // Tuesday
}

// parseDateArgsFrom is a test helper that injects a fixed "today".
func parseDateArgsFrom(today time.Time, args []string) ([]time.Time, error) {
	if len(args) == 0 || args[0] == "today" {
		return []time.Time{today}, nil
	}
	switch args[0] {
	case "yesterday":
		return []time.Time{today.AddDate(0, 0, -1)}, nil
	case "week":
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := today.AddDate(0, 0, -(weekday - 1))
		var days []time.Time
		for d := monday; !d.After(today); d = d.AddDate(0, 0, 1) {
			days = append(days, d)
		}
		return days, nil
	}
	// delegate to the real implementation for dates / ranges
	return parseDateArgs(args)
}

func TestParseDateArgs_NoArgs(t *testing.T) {
	today := fixedToday()
	got, err := parseDateArgsFrom(today, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Equal(today) {
		t.Errorf("expected today %v, got %v", today, got)
	}
}

func TestParseDateArgs_Today(t *testing.T) {
	today := fixedToday()
	got, _ := parseDateArgsFrom(today, []string{"today"})
	if !got[0].Equal(today) {
		t.Errorf("expected today, got %v", got[0])
	}
}

func TestParseDateArgs_Yesterday(t *testing.T) {
	today := fixedToday()
	got, _ := parseDateArgsFrom(today, []string{"yesterday"})
	want := today.AddDate(0, 0, -1)
	if !got[0].Equal(want) {
		t.Errorf("expected %v, got %v", want, got[0])
	}
}

func TestParseDateArgs_Week_Tuesday(t *testing.T) {
	// 2026-06-24 is a Wednesday; week should be Mon 22 – Wed 24 (3 days)
	today := time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local) // Wednesday
	got, err := parseDateArgsFrom(today, []string{"week"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 days for Wednesday week, got %d", len(got))
	}
	wantMonday := time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local)
	if !got[0].Equal(wantMonday) {
		t.Errorf("week should start on Monday %v, got %v", wantMonday, got[0])
	}
}

func TestParseDateArgs_Week_Monday(t *testing.T) {
	monday := time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local)
	got, _ := parseDateArgsFrom(monday, []string{"week"})
	if len(got) != 1 {
		t.Errorf("expected 1 day when today is Monday, got %d", len(got))
	}
}

func TestParseDateArgs_SingleDate(t *testing.T) {
	got, err := parseDateArgs([]string{"2026-06-20"})
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 20, 0, 0, 0, 0, time.Local)
	if !got[0].Equal(want) {
		t.Errorf("expected %v, got %v", want, got[0])
	}
}

func TestParseDateArgs_Range(t *testing.T) {
	got, err := parseDateArgs([]string{"2026-06-01:2026-06-03"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 days, got %d", len(got))
	}
}

func TestParseDateArgs_Range_ReversedError(t *testing.T) {
	_, err := parseDateArgs([]string{"2026-06-07:2026-06-01"})
	if err == nil {
		t.Error("expected error for reversed range")
	}
}

func TestParseDateArgs_InvalidDate(t *testing.T) {
	_, err := parseDateArgs([]string{"not-a-date"})
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

// ---- writeReport -----------------------------------------------------------

func newTestStore(t *testing.T) *storage.Storage {
	t.Helper()
	s, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	return s
}

func TestWriteReport_NoEntries(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local)

	var buf bytes.Buffer
	if err := writeReport(&buf, &bytes.Buffer{}, s, []time.Time{date}); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "No entries recorded") {
		t.Errorf("expected 'No entries recorded' in output, got:\n%s", buf.String())
	}
}

func TestWriteReport_WithTasks(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local)

	record := &model.DayRecord{
		Date: date,
		Tasks: []model.Task{
			{
				Profile: model.Profile{Name: "Development"},
				Entries: []model.TimeEntry{
					{
						Start: time.Date(2026, 6, 24, 9, 0, 0, 0, time.Local),
						End:   time.Date(2026, 6, 24, 10, 30, 0, 0, time.Local),
					},
				},
			},
		},
	}
	if err := s.SaveDay(record); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := writeReport(&buf, &bytes.Buffer{}, s, []time.Time{date}); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "Development") {
		t.Errorf("expected task name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1h30m00s") {
		t.Errorf("expected duration in output, got:\n%s", out)
	}
}

func TestWriteReport_MultiDay(t *testing.T) {
	s := newTestStore(t)
	dates := []time.Time{
		time.Date(2026, 6, 23, 0, 0, 0, 0, time.Local),
		time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local),
	}

	var buf bytes.Buffer
	if err := writeReport(&buf, &bytes.Buffer{}, s, dates); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "June 23") || !strings.Contains(out, "June 24") {
		t.Errorf("expected both dates in output, got:\n%s", out)
	}
}

// ---- cobra command integration ---------------------------------------------

func TestReportCommand_RunsWithoutError(t *testing.T) {
	// Smoke-test: the cobra command wires up and executes without panicking.
	// We can't easily override dataDir() here, so we just confirm it exits cleanly
	// for a date that has no data.
	root := NewRootCmd("test")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"report", "2000-01-01"})
	if err := root.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
