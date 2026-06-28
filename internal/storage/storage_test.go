package storage

import (
	"testing"
	"time"

	"github.com/kubeone/go-time-tracker/internal/model"
)

// roundTrip serialises a DayRecord to markdown and parses it back.
func roundTrip(t *testing.T, record *model.DayRecord) *model.DayRecord {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.SaveDay(record); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}
	got, err := s.LoadDay(record.Date)
	if err != nil {
		t.Fatalf("LoadDay: %v", err)
	}
	return got
}

func day(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.Local)
}

func ts(y, mo, d, h, min, s int) time.Time {
	return time.Date(y, time.Month(mo), d, h, min, s, 0, time.Local)
}

func TestRoundTrip_SingleTask(t *testing.T) {
	record := &model.DayRecord{
		Date: day(2026, 6, 24),
		Tasks: []model.Task{
			{
				Profile: model.Profile{Name: "Dev", Description: "Feature work"},
				Entries: []model.TimeEntry{
					{Start: ts(2026, 6, 24, 9, 0, 0), End: ts(2026, 6, 24, 10, 30, 0)},
				},
			},
		},
	}

	got := roundTrip(t, record)

	if len(got.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got.Tasks))
	}
	task := got.Tasks[0]
	if task.Profile.Name != "Dev" {
		t.Errorf("profile name: got %q, want %q", task.Profile.Name, "Dev")
	}
	if task.Profile.Description != "Feature work" {
		t.Errorf("profile desc: got %q, want %q", task.Profile.Description, "Feature work")
	}
	if len(task.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(task.Entries))
	}
	if !task.Entries[0].Start.Equal(ts(2026, 6, 24, 9, 0, 0)) {
		t.Errorf("start mismatch: got %v", task.Entries[0].Start)
	}
	if !task.Entries[0].End.Equal(ts(2026, 6, 24, 10, 30, 0)) {
		t.Errorf("end mismatch: got %v", task.Entries[0].End)
	}
}

func TestRoundTrip_MultipleTasks(t *testing.T) {
	record := &model.DayRecord{
		Date: day(2026, 6, 24),
		Tasks: []model.Task{
			{
				Profile: model.Profile{Name: "Dev"},
				Entries: []model.TimeEntry{
					{Start: ts(2026, 6, 24, 9, 0, 0), End: ts(2026, 6, 24, 10, 0, 0)},
					{Start: ts(2026, 6, 24, 11, 0, 0), End: ts(2026, 6, 24, 12, 0, 0)},
				},
			},
			{
				Profile: model.Profile{Name: "Meetings"},
				Entries: []model.TimeEntry{
					{Start: ts(2026, 6, 24, 14, 0, 0), End: ts(2026, 6, 24, 14, 30, 0)},
				},
			},
		},
	}

	got := roundTrip(t, record)

	if len(got.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got.Tasks))
	}
	if len(got.Tasks[0].Entries) != 2 {
		t.Errorf("task 0: expected 2 entries, got %d", len(got.Tasks[0].Entries))
	}
	if len(got.Tasks[1].Entries) != 1 {
		t.Errorf("task 1: expected 1 entry, got %d", len(got.Tasks[1].Entries))
	}
}

func TestRoundTrip_OngoingEntry(t *testing.T) {
	// An entry with no End should survive the round-trip as "ongoing"
	record := &model.DayRecord{
		Date: day(2026, 6, 24),
		Tasks: []model.Task{
			{
				Profile: model.Profile{Name: "Dev"},
				Entries: []model.TimeEntry{
					{Start: ts(2026, 6, 24, 9, 0, 0)}, // no End
				},
			},
		},
	}

	got := roundTrip(t, record)

	if len(got.Tasks) != 1 || len(got.Tasks[0].Entries) != 1 {
		t.Fatal("task/entry count mismatch after round-trip")
	}
	if !got.Tasks[0].Entries[0].End.IsZero() {
		t.Error("expected ongoing entry to have zero End after round-trip")
	}
}

func TestRoundTrip_EmptyRecord(t *testing.T) {
	record := &model.DayRecord{Date: day(2026, 6, 24)}
	got := roundTrip(t, record)
	if len(got.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(got.Tasks))
	}
}

func TestLoadDay_MissingFile(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	record, err := s.LoadDay(day(2000, 1, 1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(record.Tasks) != 0 {
		t.Error("expected empty record for missing file")
	}
}

func TestRoundTrip_Notes(t *testing.T) {
	record := &model.DayRecord{
		Date: day(2026, 6, 25),
		Tasks: []model.Task{
			{
				Profile: model.Profile{Name: "Dev", Description: "Feature work"},
				Entries: []model.TimeEntry{
					{Start: ts(2026, 6, 25, 9, 0, 0), End: ts(2026, 6, 25, 10, 0, 0)},
				},
				Notes: "Fixed auth bug.\nBlocked on deployment.",
			},
			{
				Profile: model.Profile{Name: "Planning"},
				Notes:   "Plan tomorrow's sprint.",
			},
		},
	}

	got := roundTrip(t, record)

	if len(got.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got.Tasks))
	}
	if got.Tasks[0].Notes != "Fixed auth bug.\nBlocked on deployment." {
		t.Errorf("task 0 notes: got %q", got.Tasks[0].Notes)
	}
	if got.Tasks[1].Notes != "Plan tomorrow's sprint." {
		t.Errorf("task 1 notes: got %q", got.Tasks[1].Notes)
	}
	if len(got.Tasks[1].Entries) != 0 {
		t.Errorf("notes-only task should have no entries")
	}
}

func TestFmtDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3*time.Hour + 5*time.Minute + 7*time.Second, "3h05m07s"},
		{2*time.Hour + 0*time.Minute + 0*time.Second, "2h00m00s"},
	}
	for _, c := range cases {
		got := FmtDuration(c.in)
		if got != c.want {
			t.Errorf("FmtDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
