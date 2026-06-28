package ui

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kubeone/go-time-tracker/internal/model"
	"github.com/kubeone/go-time-tracker/internal/storage"
	"github.com/kubeone/go-time-tracker/internal/tracker"
)

// ---- helpers ---------------------------------------------------------------

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

var (
	profDev      = model.Profile{Name: "Dev", Description: "Development"}
	profMeetings = model.Profile{Name: "Meetings", Description: "Team meetings"}
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	tr, err := tracker.New(store, time.Now())
	if err != nil {
		t.Fatalf("tracker.New: %v", err)
	}
	return NewApp(tr, store, []model.Profile{profDev, profMeetings})
}

// send dispatches any bubbletea message and returns the updated app.
func send(app *App, msg tea.Msg) *App {
	next, _ := app.Update(msg)
	return next.(*App)
}

func rune2key(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func resize(app *App, w, h int) *App {
	return send(app, tea.WindowSizeMsg{Width: w, Height: h})
}

// ---- navigation ------------------------------------------------------------

func TestNavigation_MoveDownWithJ(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('j'))
	if app.selected != 1 {
		t.Errorf("expected selected=1 after j, got %d", app.selected)
	}
}

func TestNavigation_MoveDownWithArrow(t *testing.T) {
	app := newTestApp(t)
	app = send(app, tea.KeyMsg{Type: tea.KeyDown})
	if app.selected != 1 {
		t.Errorf("expected selected=1 after down, got %d", app.selected)
	}
}

func TestNavigation_ClampsAtBottom(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('j'))
	app = send(app, rune2key('j')) // already at last profile
	if app.selected != 1 {
		t.Errorf("expected selected to clamp at 1, got %d", app.selected)
	}
}

func TestNavigation_MoveUpWithK(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('j')) // go to 1
	app = send(app, rune2key('k')) // back to 0
	if app.selected != 0 {
		t.Errorf("expected selected=0 after k, got %d", app.selected)
	}
}

func TestNavigation_MoveUpWithArrow(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('j'))
	app = send(app, tea.KeyMsg{Type: tea.KeyUp})
	if app.selected != 0 {
		t.Errorf("expected selected=0 after up arrow, got %d", app.selected)
	}
}

func TestNavigation_ClampsAtTop(t *testing.T) {
	app := newTestApp(t)
	app = send(app, tea.KeyMsg{Type: tea.KeyUp}) // already at 0
	if app.selected != 0 {
		t.Errorf("expected selected to clamp at 0, got %d", app.selected)
	}
}

// ---- start / stop / resume -------------------------------------------------

func TestStart_StartsSelectedProfile(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('s'))

	if !app.tracker.Record().IsRunning() {
		t.Error("expected timer running after s")
	}
	active := app.tracker.Record().ActiveTask()
	if active == nil || active.Profile.Name != "Dev" {
		t.Error("expected Dev to be the active task")
	}
	if app.err != "" {
		t.Errorf("unexpected error: %s", app.err)
	}
}

func TestStart_NavigateAndStart_StartsCorrectProfile(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('j')) // select Meetings
	app = send(app, rune2key('s'))

	active := app.tracker.Record().ActiveTask()
	if active == nil || active.Profile.Name != "Meetings" {
		t.Error("expected Meetings to be active after navigating to it")
	}
}

func TestStop_StopsActiveTimer(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('s'))
	app = send(app, rune2key('x'))

	if app.tracker.Record().IsRunning() {
		t.Error("expected timer stopped after x")
	}
	if app.err != "" {
		t.Errorf("unexpected error: %s", app.err)
	}
}

func TestStop_WhenNotRunning_SetsError(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('x'))

	if app.err == "" {
		t.Error("expected error when stopping with no active timer")
	}
}

func TestResume_ResumesLastTask(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('s')) // start Dev
	app = send(app, rune2key('x')) // stop
	app = send(app, rune2key('r')) // resume

	if !app.tracker.Record().IsRunning() {
		t.Error("expected timer running after resume")
	}
	active := app.tracker.Record().ActiveTask()
	if active == nil || active.Profile.Name != "Dev" {
		t.Error("expected Dev resumed")
	}
}

func TestResume_WhenNeverStarted_SetsError(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('r'))

	if app.err == "" {
		t.Error("expected error when resuming with no prior task")
	}
}

func TestError_ClearedOnNextUpdate(t *testing.T) {
	app := newTestApp(t)
	app = send(app, rune2key('x')) // triggers error
	if app.err == "" {
		t.Fatal("precondition: expected error after stop with no timer")
	}
	app = send(app, rune2key('j')) // any key clears it
	if app.err != "" {
		t.Error("expected error cleared on next update")
	}
}

// ---- quit ------------------------------------------------------------------

func TestQuit_ReturnsQuitCmd(t *testing.T) {
	app := newTestApp(t)
	_, cmd := app.Update(rune2key('q'))
	if cmd == nil {
		t.Fatal("expected non-nil cmd for q")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

// ---- window resize + view --------------------------------------------------

func TestWindowResize_UpdatesDimensions(t *testing.T) {
	app := newTestApp(t)
	app = resize(app, 120, 40)
	if app.width != 120 || app.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", app.width, app.height)
	}
}

func TestView_BeforeResize_ShowsLoading(t *testing.T) {
	app := newTestApp(t)
	if app.View() != "Loading..." {
		t.Errorf("expected 'Loading...' before resize, got %q", app.View())
	}
}

func TestView_AfterResize_RendersProfiles(t *testing.T) {
	app := newTestApp(t)
	app = resize(app, 120, 40)
	v := app.View()
	for _, want := range []string{"Time Tracker", "Dev", "Meetings"} {
		if !strings.Contains(v, want) {
			t.Errorf("expected %q in view", want)
		}
	}
}

func TestView_ShowsActiveTimer(t *testing.T) {
	app := newTestApp(t)
	app = resize(app, 120, 40)
	app = send(app, rune2key('s'))

	v := app.View()
	if !strings.Contains(v, "Active Timer") {
		t.Error("expected 'Active Timer' section in view")
	}
	if !strings.Contains(v, "Dev") {
		t.Error("expected 'Dev' in active timer view")
	}
}

func TestView_ShowsNoTimerRunning(t *testing.T) {
	app := newTestApp(t)
	app = resize(app, 120, 40)

	v := app.View()
	if !strings.Contains(v, "No timer running") {
		t.Error("expected 'No timer running' when idle")
	}
}

func TestView_ShowsErrorMessage(t *testing.T) {
	app := newTestApp(t)
	app = resize(app, 120, 40)
	app.err = "something went wrong"

	v := stripANSI(app.View())
	if !strings.Contains(v, "something went wrong") {
		t.Errorf("expected error in view, got:\n%s", v)
	}
}

// ---- tick ------------------------------------------------------------------

func TestTickMsg_UpdatesTime(t *testing.T) {
	app := newTestApp(t)
	future := app.now.Add(5 * time.Second)
	app = send(app, tickMsg(future))
	if !app.now.Equal(future) {
		t.Errorf("expected now=%v, got %v", future, app.now)
	}
}

// ---- day navigation --------------------------------------------------------

func TestSwitchDay_LeftMovesToYesterday(t *testing.T) {
	app := newTestApp(t)
	today := app.tracker.Record().Date
	app = send(app, tea.KeyMsg{Type: tea.KeyLeft})

	after := app.tracker.Record().Date
	if !sameDay(after, today.AddDate(0, 0, -1)) {
		t.Errorf("expected yesterday, got %v", after)
	}
}

func TestSwitchDay_HKeyMovesToYesterday(t *testing.T) {
	app := newTestApp(t)
	today := app.tracker.Record().Date
	app = send(app, rune2key('h'))

	if !sameDay(app.tracker.Record().Date, today.AddDate(0, 0, -1)) {
		t.Error("h key did not go to yesterday")
	}
}

func TestSwitchDay_RightDoesNotGoToFuture(t *testing.T) {
	app := newTestApp(t)
	today := app.tracker.Record().Date
	app = send(app, tea.KeyMsg{Type: tea.KeyRight})

	if !sameDay(app.tracker.Record().Date, today) {
		t.Errorf("expected to stay on today, got %v", app.tracker.Record().Date)
	}
}

func TestSwitchDay_RightReturnsFromPastDay(t *testing.T) {
	app := newTestApp(t)
	app = send(app, tea.KeyMsg{Type: tea.KeyLeft})  // go to yesterday
	app = send(app, tea.KeyMsg{Type: tea.KeyRight}) // come back

	if !sameDay(app.tracker.Record().Date, time.Now()) {
		t.Errorf("expected today after right from yesterday, got %v", app.tracker.Record().Date)
	}
}

// ---- teatest integration ---------------------------------------------------

func TestTeatest_ProfilesVisibleOnStart(t *testing.T) {
	app := newTestApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		out := stripANSI(string(b))
		return strings.Contains(out, "Dev") && strings.Contains(out, "Meetings")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(rune2key('q'))
	tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTeatest_TimerAppearsAfterStart(t *testing.T) {
	app := newTestApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	tm.Send(rune2key('s'))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		out := stripANSI(string(b))
		return strings.Contains(out, "Active Timer") && !strings.Contains(out, "No timer running")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(rune2key('q'))
	tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTeatest_ErrorShownOnBadStop(t *testing.T) {
	app := newTestApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	tm.Send(rune2key('x')) // stop with nothing running

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Error"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(rune2key('q'))
	tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTeatest_FinalModel_RecordsStartedTask(t *testing.T) {
	app := newTestApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	tm.Send(rune2key('s')) // start Dev
	tm.Send(rune2key('q'))

	fm := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	final := fm.(*App)

	task := final.tracker.Record().FindTask("Dev")
	if task == nil || len(task.Entries) == 0 {
		t.Error("expected Dev task with at least one entry in final model")
	}
}
