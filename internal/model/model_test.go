package model

import (
	"testing"
	"time"
)

func TestTimeEntry_Duration_Running(t *testing.T) {
	e := TimeEntry{Start: time.Now().Add(-5 * time.Minute)}
	d := e.Duration()
	if d < 4*time.Minute || d > 6*time.Minute {
		t.Errorf("expected ~5m, got %v", d)
	}
}

func TestTimeEntry_Duration_Completed(t *testing.T) {
	start := time.Now().Add(-10 * time.Minute)
	end := start.Add(10 * time.Minute)
	e := TimeEntry{Start: start, End: end}
	if e.Duration() != 10*time.Minute {
		t.Errorf("expected 10m, got %v", e.Duration())
	}
	if e.Running() {
		t.Error("expected Running() == false for completed entry")
	}
}

func TestTask_TotalDuration(t *testing.T) {
	now := time.Now()
	task := Task{
		Entries: []TimeEntry{
			{Start: now.Add(-30 * time.Minute), End: now.Add(-20 * time.Minute)}, // 10m
			{Start: now.Add(-10 * time.Minute), End: now},                         // 10m
		},
	}
	if task.TotalDuration() != 20*time.Minute {
		t.Errorf("expected 20m, got %v", task.TotalDuration())
	}
}

func TestTask_ActiveEntry(t *testing.T) {
	now := time.Now()
	task := Task{
		Entries: []TimeEntry{
			{Start: now.Add(-30 * time.Minute), End: now.Add(-20 * time.Minute)},
			{Start: now.Add(-5 * time.Minute)}, // running
		},
	}
	e := task.ActiveEntry()
	if e == nil {
		t.Fatal("expected active entry, got nil")
	}
	if !e.Running() {
		t.Error("expected entry to be running")
	}
}

func TestTask_ActiveEntry_None(t *testing.T) {
	now := time.Now()
	task := Task{
		Entries: []TimeEntry{
			{Start: now.Add(-30 * time.Minute), End: now.Add(-20 * time.Minute)},
		},
	}
	if task.ActiveEntry() != nil {
		t.Error("expected nil, no running entry")
	}
}

func TestDayRecord_FindTask(t *testing.T) {
	record := DayRecord{
		Tasks: []Task{
			{Profile: Profile{Name: "Dev"}},
			{Profile: Profile{Name: "Meetings"}},
		},
	}
	if record.FindTask("Dev") == nil {
		t.Error("expected to find Dev task")
	}
	if record.FindTask("Other") != nil {
		t.Error("expected nil for unknown task")
	}
}

func TestDayRecord_ActiveTask(t *testing.T) {
	now := time.Now()
	record := DayRecord{
		Tasks: []Task{
			{
				Profile: Profile{Name: "Dev"},
				Entries: []TimeEntry{{Start: now.Add(-5 * time.Minute)}},
			},
		},
	}
	if !record.IsRunning() {
		t.Error("expected IsRunning() == true")
	}
	active := record.ActiveTask()
	if active == nil || active.Profile.Name != "Dev" {
		t.Error("expected active task Dev")
	}
}
