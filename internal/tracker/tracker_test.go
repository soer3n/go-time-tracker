package tracker

import (
	"testing"
	"time"

	"github.com/kubeone/go-time-tracker/internal/model"
	"github.com/kubeone/go-time-tracker/internal/storage"
)

func newTestTracker(t *testing.T) *Tracker {
	t.Helper()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	tr, err := New(store, time.Now())
	if err != nil {
		t.Fatalf("tracker.New: %v", err)
	}
	return tr
}

var dev = model.Profile{Name: "Dev", Description: "Development"}
var meetings = model.Profile{Name: "Meetings", Description: "Team meetings"}

func TestStart_CreatesRunningEntry(t *testing.T) {
	tr := newTestTracker(t)
	if err := tr.Start(dev); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !tr.Record().IsRunning() {
		t.Error("expected timer to be running after Start")
	}
	active := tr.Record().ActiveTask()
	if active == nil || active.Profile.Name != "Dev" {
		t.Error("expected active task Dev")
	}
}

func TestStop_ClosesEntry(t *testing.T) {
	tr := newTestTracker(t)
	_ = tr.Start(dev)
	if err := tr.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if tr.Record().IsRunning() {
		t.Error("expected timer stopped after Stop")
	}
	task := tr.Record().FindTask("Dev")
	if task == nil || len(task.Entries) == 0 {
		t.Fatal("task or entries missing")
	}
	if task.Entries[0].End.IsZero() {
		t.Error("expected End to be set after Stop")
	}
}

func TestStop_WhenNotRunning_ReturnsError(t *testing.T) {
	tr := newTestTracker(t)
	if err := tr.Stop(); err == nil {
		t.Error("expected error when stopping with no active timer")
	}
}

func TestStart_StopsPreviousTask(t *testing.T) {
	tr := newTestTracker(t)
	_ = tr.Start(dev)
	_ = tr.Start(meetings)

	// Dev should now be stopped
	devTask := tr.Record().FindTask("Dev")
	if devTask == nil {
		t.Fatal("Dev task missing")
	}
	if devTask.ActiveEntry() != nil {
		t.Error("expected Dev to be stopped after starting Meetings")
	}

	// Meetings should be running
	active := tr.Record().ActiveTask()
	if active == nil || active.Profile.Name != "Meetings" {
		t.Error("expected Meetings to be the active task")
	}
}

func TestRestart_StartsNewEntryOnSameTask(t *testing.T) {
	tr := newTestTracker(t)
	_ = tr.Start(dev)
	_ = tr.Stop()

	if err := tr.Restart(); err != nil {
		t.Fatalf("Restart: %v", err)
	}

	task := tr.Record().FindTask("Dev")
	if task == nil {
		t.Fatal("Dev task missing after restart")
	}
	if len(task.Entries) != 2 {
		t.Errorf("expected 2 entries after restart, got %d", len(task.Entries))
	}
	if task.ActiveEntry() == nil {
		t.Error("expected task to be running after Restart")
	}
}

func TestRestart_WhenRunning_RestartsTimer(t *testing.T) {
	tr := newTestTracker(t)
	_ = tr.Start(dev)

	if err := tr.Restart(); err != nil {
		t.Fatalf("Restart while running: %v", err)
	}

	task := tr.Record().FindTask("Dev")
	if task == nil {
		t.Fatal("Dev task missing")
	}
	// first entry should be closed, second running
	if len(task.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(task.Entries))
	}
	if task.Entries[0].End.IsZero() {
		t.Error("first entry should be closed")
	}
	if !task.Entries[1].End.IsZero() {
		t.Error("second entry should still be running")
	}
}

func TestRestart_WhenNeverStarted_ReturnsError(t *testing.T) {
	tr := newTestTracker(t)
	if err := tr.Restart(); err == nil {
		t.Error("expected error when restarting with no prior task")
	}
}

func TestStart_AppendsEntriesToExistingTask(t *testing.T) {
	tr := newTestTracker(t)
	_ = tr.Start(dev)
	_ = tr.Stop()
	_ = tr.Start(meetings)
	_ = tr.Stop()
	_ = tr.Start(dev) // second session on Dev

	task := tr.Record().FindTask("Dev")
	if task == nil {
		t.Fatal("Dev task missing")
	}
	if len(task.Entries) != 2 {
		t.Errorf("expected 2 entries on Dev, got %d", len(task.Entries))
	}
}

func TestSwitchDay_StopsRunningTimer(t *testing.T) {
	tr := newTestTracker(t)
	_ = tr.Start(dev)

	yesterday := time.Now().AddDate(0, 0, -1)
	if err := tr.SwitchDay(yesterday); err != nil {
		t.Fatalf("SwitchDay: %v", err)
	}

	// After switching, record should be for yesterday and not running
	if tr.Record().IsRunning() {
		t.Error("expected no running timer after switching day")
	}
}
