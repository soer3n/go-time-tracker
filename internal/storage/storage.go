package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kubeone/go-time-tracker/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	configFile    = "config.yaml"
	daysDir       = "days"
	dateLayout    = "2006-01-02"
	timeLayout    = "15:04:05"
	defaultConfig = `profiles:
  - name: "Development"
    description: "Feature development and bug fixes"
  - name: "Meetings"
    description: "Team meetings and standups"
  - name: "Code Review"
    description: "Reviewing pull requests"
  - name: "Other"
    description: "Miscellaneous tasks"
`
)

type Storage struct {
	baseDir string
}

func New(baseDir string) (*Storage, error) {
	if err := os.MkdirAll(filepath.Join(baseDir, daysDir), 0755); err != nil {
		return nil, err
	}
	return &Storage{baseDir: baseDir}, nil
}

func (s *Storage) ConfigPath() string {
	return filepath.Join(s.baseDir, configFile)
}

func (s *Storage) LoadConfig() (*model.Config, error) {
	path := s.ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.DataDir == "" {
		cfg.DataDir = s.baseDir
	}
	return &cfg, nil
}

func (s *Storage) SaveConfig(cfg *model.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(s.ConfigPath(), data, 0644)
}

func (s *Storage) dayPath(date time.Time) string {
	return filepath.Join(s.baseDir, daysDir, date.Format(dateLayout)+".md")
}

func (s *Storage) DayPath(date time.Time) string {
	return s.dayPath(date)
}

func (s *Storage) LoadDay(date time.Time) (*model.DayRecord, error) {
	path := s.dayPath(date)
	record := &model.DayRecord{Date: date}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return record, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseMarkdown(record, f)
}

func (s *Storage) SaveDay(record *model.DayRecord) error {
	path := s.dayPath(record.Date)
	content := formatMarkdown(record)
	return os.WriteFile(path, []byte(content), 0644)
}

func (s *Storage) ListDays() ([]time.Time, error) {
	pattern := filepath.Join(s.baseDir, daysDir, "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var days []time.Time
	for _, m := range matches {
		base := filepath.Base(m)
		base = strings.TrimSuffix(base, ".md")
		t, err := time.ParseInLocation(dateLayout, base, time.Local)
		if err != nil {
			continue
		}
		days = append(days, t)
	}
	return days, nil
}

// formatMarkdown serialises a DayRecord to markdown.
func formatMarkdown(record *model.DayRecord) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", record.Date.Format("2006-01-02 Monday")))

	for _, task := range record.Tasks {
		if len(task.Entries) == 0 && task.Notes == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s\n", task.Profile.Name))
		if task.Profile.Description != "" {
			sb.WriteString(fmt.Sprintf("> %s\n", task.Profile.Description))
		}
		sb.WriteString("\n")
		for _, e := range task.Entries {
			endStr := "ongoing"
			if !e.End.IsZero() {
				endStr = e.End.Format(timeLayout)
			}
			sb.WriteString(fmt.Sprintf("- %s – %s (%s)\n",
				e.Start.Format(timeLayout),
				endStr,
				fmtDuration(e.Duration()),
			))
		}
		if len(task.Entries) > 0 {
			sb.WriteString(fmt.Sprintf("\n**Total: %s**\n", fmtDuration(task.TotalDuration())))
		}
		if task.Notes != "" {
			sb.WriteString("\n")
			for _, line := range strings.Split(task.Notes, "\n") {
				sb.WriteString(line + "\n")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

var (
	reSection = regexp.MustCompile(`^## (.+)$`)
	reDesc    = regexp.MustCompile(`^> (.+)$`)
	reEntry   = regexp.MustCompile(`^- (\d{2}:\d{2}:\d{2}) – (\S+)`)
	reTotal   = regexp.MustCompile(`^\*\*Total:`)
)

func parseMarkdown(record *model.DayRecord, f *os.File) (*model.DayRecord, error) {
	scanner := bufio.NewScanner(f)
	dateStr := record.Date.Format(dateLayout)

	var current *model.Task
	inNotes := false

	for scanner.Scan() {
		line := scanner.Text()

		if m := reSection.FindStringSubmatch(line); m != nil {
			if current != nil {
				current.Notes = strings.TrimRight(current.Notes, "\n")
			}
			task := model.Task{Profile: model.Profile{Name: m[1]}}
			record.Tasks = append(record.Tasks, task)
			current = &record.Tasks[len(record.Tasks)-1]
			inNotes = false
			continue
		}

		if current == nil {
			continue
		}

		if reTotal.MatchString(line) {
			inNotes = true
			continue
		}

		if inNotes {
			if line != "" {
				current.Notes += line + "\n"
			}
			continue
		}

		if m := reDesc.FindStringSubmatch(line); m != nil {
			current.Profile.Description = m[1]
			continue
		}

		if m := reEntry.FindStringSubmatch(line); m != nil {
			startT, err := time.ParseInLocation(dateLayout+" "+timeLayout, dateStr+" "+m[1], time.Local)
			if err != nil {
				continue
			}
			entry := model.TimeEntry{Start: startT}
			if m[2] != "ongoing" {
				endT, err := time.ParseInLocation(dateLayout+" "+timeLayout, dateStr+" "+m[2], time.Local)
				if err == nil {
					entry.End = endT
				}
			}
			current.Entries = append(current.Entries, entry)
			continue
		}

		// Non-special, non-blank line before any entries → pre-task note.
		if line != "" {
			current.Notes += line + "\n"
		}
	}

	if current != nil {
		current.Notes = strings.TrimRight(current.Notes, "\n")
	}

	return record, scanner.Err()
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func FmtDuration(d time.Duration) string {
	return fmtDuration(d)
}
