package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kubeone/go-time-tracker/internal/model"
	"github.com/kubeone/go-time-tracker/internal/storage"
	"github.com/kubeone/go-time-tracker/internal/tracker"
)

// tickMsg fires every second to update the running timer.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ---- UI modes --------------------------------------------------------------

const (
	modeNormal      = iota
	modeNotes       // editing notes for a profile
	modeEntrySelect // browsing entries to pick one to edit
	modeEntryEdit   // editing start/end time of a selected entry
)

// entryRef points to a specific time entry within the day record.
type entryRef struct {
	taskIdx  int
	entryIdx int
}

// ---- styles ----------------------------------------------------------------

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}
	highlight = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#EE6FF8"}
	accent    = lipgloss.AdaptiveColor{Light: "#0066FF", Dark: "#74B2FF"}
	green     = lipgloss.AdaptiveColor{Light: "#02A552", Dark: "#04B575"}
	red       = lipgloss.AdaptiveColor{Light: "#D9534F", Dark: "#FF5F57"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#383838", Dark: "#DDDDDD"})

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#D9EBFF", Dark: "#1A3A5C"}).
			Foreground(accent).
			Bold(true).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	runningStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	durationStyle = lipgloss.NewStyle().
			Foreground(subtle)

	timerStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true).
			Padding(1, 2)

	timerLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#383838", Dark: "#DDDDDD"}).
			Bold(true).
			Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle)

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accent)

	errorStyle = lipgloss.NewStyle().
			Foreground(red)

	notePreviewStyle = lipgloss.NewStyle().
				Foreground(subtle).
				Italic(true)

	fieldLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#383838", Dark: "#DDDDDD"}).
			Padding(0, 1)
)

// ---- model -----------------------------------------------------------------

type App struct {
	tracker  *tracker.Tracker
	store    *storage.Storage
	profiles []model.Profile
	selected int
	width    int
	height   int
	err      string
	now      time.Time

	mode int

	// notes state
	noteTA        textarea.Model
	notingProfile model.Profile

	// entry-select state
	entryList     []entryRef
	entrySelected int

	// entry-edit state
	editStart textinput.Model
	editEnd   textinput.Model
	editFocus int // 0 = start, 1 = end
}

func NewApp(tr *tracker.Tracker, store *storage.Storage, profiles []model.Profile) *App {
	return &App{
		tracker:  tr,
		store:    store,
		profiles: profiles,
		now:      time.Now(),
	}
}

func (a *App) Init() tea.Cmd {
	return tick()
}

// ---- Update ----------------------------------------------------------------

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	a.err = ""
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		a.now = time.Time(msg)
		if err := a.tracker.PollReload(); err != nil {
			a.err = err.Error()
		}
		cmds = append(cmds, tick())

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		switch a.mode {
		case modeNotes:
			return a.updateNotes(msg, cmds)
		case modeEntrySelect:
			return a.updateEntrySelect(msg, cmds)
		case modeEntryEdit:
			return a.updateEntryEdit(msg, cmds)
		default:
			return a.updateNormal(msg, cmds)
		}
	}

	// Non-key messages: forward to active sub-component.
	switch a.mode {
	case modeNotes:
		var taCmd tea.Cmd
		a.noteTA, taCmd = a.noteTA.Update(msg)
		cmds = append(cmds, taCmd)
	case modeEntryEdit:
		var cmd tea.Cmd
		if a.editFocus == 0 {
			a.editStart, cmd = a.editStart.Update(msg)
		} else {
			a.editEnd, cmd = a.editEnd.Update(msg)
		}
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) updateNormal(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit

	case "up", "k":
		if a.selected > 0 {
			a.selected--
		}

	case "down", "j":
		if a.selected < len(a.profiles)-1 {
			a.selected++
		}

	case "s":
		if a.selected < len(a.profiles) {
			if err := a.tracker.Start(a.profiles[a.selected]); err != nil {
				a.err = err.Error()
			}
		}

	case "x":
		if err := a.tracker.Stop(); err != nil {
			a.err = err.Error()
		}

	case "r":
		if err := a.tracker.Restart(); err != nil {
			a.err = err.Error()
		}

	case "left", "h":
		day := a.tracker.Record().Date.AddDate(0, 0, -1)
		if err := a.tracker.SwitchDay(day); err != nil {
			a.err = err.Error()
		}

	case "right", "l":
		next := a.tracker.Record().Date.AddDate(0, 0, 1)
		if !next.After(time.Now()) {
			if err := a.tracker.SwitchDay(next); err != nil {
				a.err = err.Error()
			}
		}

	case "n":
		if a.selected < len(a.profiles) {
			a.enterNoteMode()
		}

	case "e":
		record := a.tracker.Record()
		list := buildEntryList(record)
		if len(list) == 0 {
			a.err = "no entries to edit"
		} else {
			a.entryList = list
			a.entrySelected = len(list) - 1 // start at the most recent entry
			a.mode = modeEntrySelect
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) updateNotes(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+s":
		notes := strings.TrimSpace(a.noteTA.Value())
		if err := a.tracker.SetNotes(a.notingProfile, notes); err != nil {
			a.err = err.Error()
		}
		a.mode = modeNormal
		return a, nil
	}
	var taCmd tea.Cmd
	a.noteTA, taCmd = a.noteTA.Update(msg)
	return a, tea.Batch(append(cmds, taCmd)...)
}

func (a *App) updateEntrySelect(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		a.mode = modeNormal

	case "up", "k":
		if a.entrySelected > 0 {
			a.entrySelected--
		}

	case "down", "j":
		if a.entrySelected < len(a.entryList)-1 {
			a.entrySelected++
		}

	case "enter", "e":
		a.enterEntryEdit()
	}

	return a, tea.Batch(cmds...)
}

func (a *App) updateEntryEdit(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = modeEntrySelect
		return a, tea.Batch(cmds...)

	case "enter":
		if err := a.saveEntryEdit(); err != nil {
			a.err = err.Error()
		} else {
			// Rebuild list in case durations changed, stay in entry select.
			a.entryList = buildEntryList(a.tracker.Record())
			a.mode = modeEntrySelect
		}
		return a, tea.Batch(cmds...)

	case "tab", "shift+tab":
		var focusCmd tea.Cmd
		if a.editFocus == 0 {
			a.editStart.Blur()
			a.editFocus = 1
			focusCmd = a.editEnd.Focus()
		} else {
			a.editEnd.Blur()
			a.editFocus = 0
			focusCmd = a.editStart.Focus()
		}
		return a, tea.Batch(append(cmds, focusCmd)...)
	}

	// Forward all other keys to the focused input.
	var cmd tea.Cmd
	if a.editFocus == 0 {
		a.editStart, cmd = a.editStart.Update(msg)
	} else {
		a.editEnd, cmd = a.editEnd.Update(msg)
	}
	return a, tea.Batch(append(cmds, cmd)...)
}

// ---- mode transitions ------------------------------------------------------

func (a *App) enterNoteMode() {
	profile := a.profiles[a.selected]
	rightWidth := a.width - (a.width/2 - 2) - 4

	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.SetWidth(rightWidth - 2)
	ta.SetHeight(max(4, a.height-8))
	ta.Placeholder = "Add notes here..."
	ta.CharLimit = 0

	record := a.tracker.Record()
	task := record.FindTask(profile.Name)
	if task != nil && task.Notes != "" {
		ta.SetValue(task.Notes)
	}
	ta.Focus()

	a.noteTA = ta
	a.notingProfile = profile
	a.mode = modeNotes
}

func (a *App) enterEntryEdit() {
	if len(a.entryList) == 0 {
		return
	}
	ref := a.entryList[a.entrySelected]
	record := a.tracker.Record()
	entry := record.Tasks[ref.taskIdx].Entries[ref.entryIdx]

	startIn := textinput.New()
	startIn.Placeholder = "HH:MM:SS"
	startIn.CharLimit = 8
	startIn.Width = 10
	startIn.SetValue(entry.Start.Format("15:04:05"))
	startIn.CursorEnd()
	cmd := startIn.Focus()
	_ = cmd

	endIn := textinput.New()
	endIn.Width = 10
	endIn.CharLimit = 8
	if entry.End.IsZero() {
		endIn.Placeholder = "ongoing"
	} else {
		endIn.SetValue(entry.End.Format("15:04:05"))
		endIn.CursorEnd()
		endIn.Placeholder = "HH:MM:SS"
	}

	a.editStart = startIn
	a.editEnd = endIn
	a.editFocus = 0
	a.mode = modeEntryEdit
}

func (a *App) saveEntryEdit() error {
	ref := a.entryList[a.entrySelected]
	record := a.tracker.Record()
	dateStr := record.Date.Format("2006-01-02")

	start, err := parseTimeInput(dateStr, a.editStart.Value())
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	var end time.Time
	if endVal := strings.TrimSpace(a.editEnd.Value()); endVal != "" {
		end, err = parseTimeInput(dateStr, endVal)
		if err != nil {
			return fmt.Errorf("end: %w", err)
		}
		if !end.After(start) {
			return fmt.Errorf("end must be after start")
		}
	}

	return a.tracker.UpdateEntry(ref.taskIdx, ref.entryIdx, start, end)
}

// ---- View ------------------------------------------------------------------

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	record := a.tracker.Record()
	date := record.Date

	// header bar
	isToday := sameDay(date, a.now)
	dateLabel := date.Format("Monday, January 2, 2006")
	if isToday {
		dateLabel += " (today)"
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		titleStyle.Render("Time Tracker"),
		headerStyle.Render("  "+dateLabel),
	)
	navHint := helpStyle.Render("← / → switch day")
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top,
		header,
		strings.Repeat(" ", max(0, a.width-lipgloss.Width(header)-lipgloss.Width(navHint)-2)),
		navHint,
	)

	leftWidth := a.width/2 - 2
	rightWidth := a.width - leftWidth - 5

	var leftContent, rightContent string
	var rightActive bool

	switch a.mode {
	case modeNotes:
		leftContent = a.renderTaskList(record, leftWidth)
		rightContent = a.renderNoteEditor(rightWidth)
		rightActive = true

	case modeEntrySelect:
		leftContent = a.renderEntryList(record, leftWidth)
		rightContent = a.renderEntryDetail(record, rightWidth)
		rightActive = false

	case modeEntryEdit:
		leftContent = a.renderEntryList(record, leftWidth)
		rightContent = a.renderEntryEditForm(record, rightWidth)
		rightActive = true

	default:
		leftContent = a.renderTaskList(record, leftWidth)
		rightContent = a.renderTimer(record, rightWidth)
		rightActive = record.IsRunning()
	}

	leftPanel := borderStyle.Width(leftWidth).Render(leftContent)
	var rightPanel string
	if rightActive {
		rightPanel = activeBorderStyle.Width(rightWidth).Render(rightContent)
	} else {
		rightPanel = borderStyle.Width(rightWidth).Render(rightContent)
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)

	// help bar
	help := a.helpLine()
	var errLine string
	if a.err != "" {
		errLine = "\n" + errorStyle.Render("Error: "+a.err)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		headerRow,
		"",
		panels,
		"",
		help+errLine,
	)
}

func (a *App) helpLine() string {
	switch a.mode {
	case modeNotes:
		return helpStyle.Render("[enter] new line  [esc / ctrl+s] save & close")
	case modeEntrySelect:
		return helpStyle.Render("↑↓/jk select  [enter] edit  [esc] back")
	case modeEntryEdit:
		return helpStyle.Render("[tab] switch field  [enter] save  [esc] cancel")
	default:
		return helpStyle.Render("↑↓/jk select  [s] start  [x] stop  [r] resume  [e] edit entries  [n] notes  [q] quit")
	}
}

// ---- render helpers --------------------------------------------------------

func (a *App) renderTaskList(record *model.DayRecord, width int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Tasks") + "\n\n")

	totalDay := time.Duration(0)

	for _, task := range record.Tasks {
		if len(task.Entries) == 0 {
			continue
		}
		d := task.TotalDuration()
		totalDay += d
		active := task.ActiveEntry() != nil
		name := task.Profile.Name
		dur := storage.FmtDuration(d)
		if active {
			name = "● " + name
		} else {
			name = "  " + name
		}
		line := fmt.Sprintf("%-*s %s", width-12, name, dur)
		if active {
			sb.WriteString(runningStyle.Render(line) + "\n")
		} else {
			sb.WriteString(durationStyle.Render(line) + "\n")
		}
		if task.Notes != "" {
			preview := firstLine(task.Notes)
			if len(preview) > width-6 {
				preview = preview[:width-9] + "..."
			}
			sb.WriteString(notePreviewStyle.Padding(0, 3).Render("↳ "+preview) + "\n")
		}
	}

	if totalDay > 0 {
		sb.WriteString("\n" + durationStyle.Render(fmt.Sprintf("Day total: %s", storage.FmtDuration(totalDay))) + "\n")
	}

	sb.WriteString("\n" + headerStyle.Render("Profiles") + "\n\n")

	for i, p := range a.profiles {
		label := fmt.Sprintf("%-*s", width-4, p.Name)
		if i == a.selected {
			sb.WriteString(selectedStyle.Render(label) + "\n")
		} else {
			sb.WriteString(normalStyle.Render(label) + "\n")
		}
		if p.Description != "" {
			sb.WriteString(durationStyle.Padding(0, 2).Render(p.Description) + "\n")
		}
	}

	return sb.String()
}

func (a *App) renderTimer(record *model.DayRecord, width int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Active Timer") + "\n\n")

	activeTask := record.ActiveTask()
	if activeTask == nil {
		sb.WriteString(durationStyle.Render("No timer running.\n\nSelect a profile and press [s] to start."))

		if a.selected < len(a.profiles) {
			task := record.FindTask(a.profiles[a.selected].Name)
			if task != nil && task.Notes != "" {
				sb.WriteString("\n\n" + headerStyle.Render("Notes") + "\n\n")
				sb.WriteString(durationStyle.Padding(0, 2).Render(task.Notes) + "\n")
			}
		}
		return sb.String()
	}

	entry := activeTask.ActiveEntry()
	elapsed := time.Since(entry.Start).Round(time.Second)

	sb.WriteString(timerLabelStyle.Render(activeTask.Profile.Name) + "\n")
	if activeTask.Profile.Description != "" {
		sb.WriteString(durationStyle.Padding(0, 2).Render(activeTask.Profile.Description) + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(timerStyle.Render(formatElapsed(elapsed)) + "\n")
	sb.WriteString(durationStyle.Padding(0, 2).Render("Started: "+entry.Start.Format("15:04:05")) + "\n")

	total := activeTask.TotalDuration()
	if total > elapsed {
		sb.WriteString(durationStyle.Padding(0, 2).Render(fmt.Sprintf("Session total: %s", storage.FmtDuration(total))) + "\n")
	}

	if activeTask.Notes != "" {
		sb.WriteString("\n" + headerStyle.Render("Notes") + "\n\n")
		sb.WriteString(durationStyle.Padding(0, 2).Render(activeTask.Notes) + "\n")
	}

	return sb.String()
}

func (a *App) renderNoteEditor(width int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Notes – "+a.notingProfile.Name) + "\n\n")
	a.noteTA.SetWidth(width - 2)
	sb.WriteString(a.noteTA.View())
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("[enter] new line  [esc / ctrl+s] save & close"))
	return sb.String()
}

func (a *App) renderEntryList(record *model.DayRecord, width int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Entries") + "\n\n")

	idx := 0
	for _, task := range record.Tasks {
		if len(task.Entries) == 0 {
			continue
		}
		sb.WriteString(durationStyle.Padding(0, 1).Render(task.Profile.Name) + "\n")
		for _, entry := range task.Entries {
			endStr := "ongoing"
			if !entry.End.IsZero() {
				endStr = entry.End.Format("15:04:05")
			}
			line := fmt.Sprintf("%s – %s  %s",
				entry.Start.Format("15:04:05"),
				endStr,
				storage.FmtDuration(entry.Duration()),
			)
			if idx == a.entrySelected {
				sb.WriteString(selectedStyle.Render(line) + "\n")
			} else {
				sb.WriteString(durationStyle.Padding(0, 2).Render(line) + "\n")
			}
			idx++
		}
		sb.WriteString("\n")
	}

	if idx == 0 {
		sb.WriteString(durationStyle.Render("No entries yet."))
	}

	return sb.String()
}

func (a *App) renderEntryDetail(record *model.DayRecord, width int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Entry") + "\n\n")

	if len(a.entryList) == 0 {
		return sb.String()
	}
	ref := a.entryList[a.entrySelected]
	task := record.Tasks[ref.taskIdx]
	entry := task.Entries[ref.entryIdx]

	sb.WriteString(headerStyle.Padding(0, 1).Render(task.Profile.Name) + "\n\n")

	endStr := "ongoing"
	if !entry.End.IsZero() {
		endStr = entry.End.Format("15:04:05")
	}

	sb.WriteString(fieldLabelStyle.Render("Start    ") + durationStyle.Render(entry.Start.Format("15:04:05")) + "\n")
	sb.WriteString(fieldLabelStyle.Render("End      ") + durationStyle.Render(endStr) + "\n")
	sb.WriteString(fieldLabelStyle.Render("Duration ") + durationStyle.Render(storage.FmtDuration(entry.Duration())) + "\n")
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[enter] edit this entry"))

	return sb.String()
}

func (a *App) renderEntryEditForm(record *model.DayRecord, width int) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Edit Entry") + "\n\n")

	if len(a.entryList) == 0 {
		return sb.String()
	}
	ref := a.entryList[a.entrySelected]
	task := record.Tasks[ref.taskIdx]

	sb.WriteString(headerStyle.Padding(0, 1).Render(task.Profile.Name) + "\n\n")
	sb.WriteString(fieldLabelStyle.Render("Start  ") + a.editStart.View() + "\n")
	sb.WriteString(fieldLabelStyle.Render("End    ") + a.editEnd.View() + "\n")
	sb.WriteString("\n")
	sb.WriteString(durationStyle.Padding(0, 1).Render("HH:MM or HH:MM:SS  ·  leave End blank to keep ongoing") + "\n")

	return sb.String()
}

// ---- helpers ---------------------------------------------------------------

func buildEntryList(record *model.DayRecord) []entryRef {
	var list []entryRef
	for ti, task := range record.Tasks {
		for ei := range task.Entries {
			list = append(list, entryRef{taskIdx: ti, entryIdx: ei})
		}
	}
	return list
}

func parseTimeInput(dateStr, s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr+" "+s, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+s, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("use HH:MM or HH:MM:SS")
}

func formatElapsed(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// Suppress unused import warning for highlight style (reserved for future use).
var _ = highlight
