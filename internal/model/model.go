package model

import "time"

type Profile struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Config struct {
	Profiles []Profile `yaml:"profiles"`
	DataDir  string    `yaml:"data_dir,omitempty"`
}

type TimeEntry struct {
	Start time.Time
	End   time.Time // zero if still running
}

func (e TimeEntry) Duration() time.Duration {
	if e.End.IsZero() {
		return time.Since(e.Start)
	}
	return e.End.Sub(e.Start)
}

func (e TimeEntry) Running() bool {
	return e.End.IsZero()
}

type Task struct {
	Profile  Profile
	Entries  []TimeEntry
	Notes    string
}

func (t *Task) TotalDuration() time.Duration {
	var total time.Duration
	for _, e := range t.Entries {
		total += e.Duration()
	}
	return total
}

func (t *Task) ActiveEntry() *TimeEntry {
	for i := range t.Entries {
		if t.Entries[i].Running() {
			return &t.Entries[i]
		}
	}
	return nil
}

type DayRecord struct {
	Date  time.Time
	Tasks []Task
}

func (d *DayRecord) IsRunning() bool {
	for i := range d.Tasks {
		if d.Tasks[i].ActiveEntry() != nil {
			return true
		}
	}
	return false
}

func (d *DayRecord) ActiveTask() *Task {
	for i := range d.Tasks {
		if d.Tasks[i].ActiveEntry() != nil {
			return &d.Tasks[i]
		}
	}
	return nil
}

func (d *DayRecord) FindTask(profileName string) *Task {
	for i := range d.Tasks {
		if d.Tasks[i].Profile.Name == profileName {
			return &d.Tasks[i]
		}
	}
	return nil
}
