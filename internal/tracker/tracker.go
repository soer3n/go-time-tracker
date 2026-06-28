package tracker

import (
	"fmt"
	"os"
	"time"

	"github.com/kubeone/go-time-tracker/internal/model"
	"github.com/kubeone/go-time-tracker/internal/storage"
)

type Tracker struct {
	store       *storage.Storage
	record      *model.DayRecord
	date        time.Time
	lastProfile *model.Profile // most recently started profile, for Restart after Stop
	lastSaved   time.Time      // mtime of the file when we last wrote or read it
}

func New(store *storage.Storage, date time.Time) (*Tracker, error) {
	record, err := store.LoadDay(date)
	if err != nil {
		return nil, err
	}
	tr := &Tracker{store: store, record: record, date: date}
	tr.lastSaved = fileMtime(store.DayPath(date))
	// Restore lastProfile from any existing active or most-recent task
	if active := record.ActiveTask(); active != nil {
		p := active.Profile
		tr.lastProfile = &p
	}
	return tr, nil
}

func (t *Tracker) Record() *model.DayRecord {
	return t.record
}

func (t *Tracker) Start(profile model.Profile) error {
	if t.record.IsRunning() {
		if err := t.stopActive(); err != nil {
			return err
		}
	}

	task := t.record.FindTask(profile.Name)
	if task == nil {
		t.record.Tasks = append(t.record.Tasks, model.Task{Profile: profile})
		task = &t.record.Tasks[len(t.record.Tasks)-1]
	}

	task.Entries = append(task.Entries, model.TimeEntry{Start: time.Now()})
	p := profile
	t.lastProfile = &p
	return t.save()
}

func (t *Tracker) Stop() error {
	if !t.record.IsRunning() {
		return fmt.Errorf("no active timer")
	}
	if err := t.stopActive(); err != nil {
		return err
	}
	return t.save()
}

func (t *Tracker) Restart() error {
	var profile model.Profile
	if active := t.record.ActiveTask(); active != nil {
		profile = active.Profile
		if err := t.stopActive(); err != nil {
			return err
		}
	} else if t.lastProfile != nil {
		profile = *t.lastProfile
	} else {
		return fmt.Errorf("no task was running")
	}
	return t.Start(profile)
}

func (t *Tracker) SwitchDay(date time.Time) error {
	if t.record.IsRunning() {
		if err := t.stopActive(); err != nil {
			return err
		}
		if err := t.save(); err != nil {
			return err
		}
	}

	record, err := t.store.LoadDay(date)
	if err != nil {
		return err
	}
	t.record = record
	t.date = date
	t.lastSaved = fileMtime(t.store.DayPath(date))
	return nil
}

func (t *Tracker) UpdateEntry(taskIdx, entryIdx int, start, end time.Time) error {
	if taskIdx >= len(t.record.Tasks) {
		return fmt.Errorf("task index out of range")
	}
	task := &t.record.Tasks[taskIdx]
	if entryIdx >= len(task.Entries) {
		return fmt.Errorf("entry index out of range")
	}
	task.Entries[entryIdx].Start = start
	task.Entries[entryIdx].End = end
	return t.save()
}

func (t *Tracker) SetNotes(profile model.Profile, notes string) error {
	task := t.record.FindTask(profile.Name)
	if task == nil {
		t.record.Tasks = append(t.record.Tasks, model.Task{Profile: profile})
		task = &t.record.Tasks[len(t.record.Tasks)-1]
	}
	task.Notes = notes
	return t.save()
}

// PollReload reloads the day record from disk if the file has been modified
// since the last time the tracker wrote or read it. Safe to call on every tick.
func (t *Tracker) PollReload() error {
	mtime := fileMtime(t.store.DayPath(t.date))
	if mtime.IsZero() || !mtime.After(t.lastSaved) {
		return nil
	}
	record, err := t.store.LoadDay(t.date)
	if err != nil {
		return err
	}
	t.record = record
	if active := record.ActiveTask(); active != nil {
		p := active.Profile
		t.lastProfile = &p
	}
	// Re-save so formatMarkdown rewrites stale duration brackets and Totals
	// that resulted from manual time edits.
	return t.save()
}

func (t *Tracker) save() error {
	err := t.store.SaveDay(t.record)
	if err == nil {
		t.lastSaved = fileMtime(t.store.DayPath(t.date))
	}
	return err
}

func (t *Tracker) stopActive() error {
	for i := range t.record.Tasks {
		e := t.record.Tasks[i].ActiveEntry()
		if e != nil {
			e.End = time.Now()
			return nil
		}
	}
	return nil
}

func fileMtime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
