package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/kubeone/go-time-tracker/internal/storage"
)

var (
	rptSubtle  = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}
	rptAccent  = lipgloss.AdaptiveColor{Light: "#0066FF", Dark: "#74B2FF"}

	rptTitle   = lipgloss.NewStyle().Bold(true).Foreground(rptAccent)
	rptHeader  = lipgloss.NewStyle().Bold(true)
	rptDim     = lipgloss.NewStyle().Foreground(rptSubtle)
	rptSep     = lipgloss.NewStyle().Foreground(rptSubtle)
)

func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report [date|range]",
		Short: "Show a time report",
		Long: `Show a time report for a specific date or date range.

Examples:
  gtt report              # today
  gtt report today        # today
  gtt report yesterday    # yesterday
  gtt report 2026-06-20   # specific date
  gtt report week         # current week (Mon–today)
  gtt report 2026-06-01:2026-06-07  # date range
`,
		RunE: runReport,
	}
	return cmd
}

func runReport(cmd *cobra.Command, args []string) error {
	store, err := storage.New(dataDir())
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	dates, err := parseDateArgs(args)
	if err != nil {
		return err
	}
	return writeReport(cmd.OutOrStdout(), cmd.ErrOrStderr(), store, dates)
}

func writeReport(w io.Writer, errW io.Writer, store *storage.Storage, dates []time.Time) error {
	for _, date := range dates {
		record, err := store.LoadDay(date)
		if err != nil {
			fmt.Fprintf(errW, "warning: could not load %s: %v\n", date.Format("2006-01-02"), err)
			continue
		}

		fmt.Fprintln(w, rptTitle.Render(fmt.Sprintf("\n%s", date.Format("Monday, January 2, 2006"))))
		fmt.Fprintln(w, rptSep.Render(strings.Repeat("─", 50)))

		if len(record.Tasks) == 0 {
			fmt.Fprintln(w, rptDim.Render("  No entries recorded."))
			continue
		}

		var dayTotal time.Duration
		for _, task := range record.Tasks {
			total := task.TotalDuration()
			dayTotal += total
			fmt.Fprintf(w, "  %-30s  %s\n", rptHeader.Render(task.Profile.Name), storage.FmtDuration(total))
			for _, e := range task.Entries {
				endStr := "ongoing"
				if !e.End.IsZero() {
					endStr = e.End.Format("15:04:05")
				}
				fmt.Fprintf(w, "    %s  %s – %s  (%s)\n",
					rptDim.Render("·"),
					e.Start.Format("15:04:05"),
					endStr,
					storage.FmtDuration(e.Duration()),
				)
			}
			if task.Notes != "" {
				lines := strings.Split(task.Notes, "\n")
				fmt.Fprintf(w, "    %s  %s\n", rptDim.Render("↳"), rptDim.Render(lines[0]))
				for _, line := range lines[1:] {
					fmt.Fprintf(w, "       %s\n", rptDim.Render(line))
				}
			}
		}
		fmt.Fprintln(w, rptSep.Render(strings.Repeat("─", 50)))
		fmt.Fprintf(w, "  %-30s  %s\n", rptHeader.Render("Total"), storage.FmtDuration(dayTotal))
	}
	fmt.Fprintln(w)
	return nil
}

func parseDateArgs(args []string) ([]time.Time, error) {
	today := truncateDay(time.Now())

	if len(args) == 0 || args[0] == "today" {
		return []time.Time{today}, nil
	}

	switch args[0] {
	case "yesterday":
		return []time.Time{today.AddDate(0, 0, -1)}, nil

	case "week":
		// Monday of the current week through today
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

	// range: "2026-06-01:2026-06-07"
	if strings.Contains(args[0], ":") {
		parts := strings.SplitN(args[0], ":", 2)
		start, err := parseDate(parts[0])
		if err != nil {
			return nil, err
		}
		end, err := parseDate(parts[1])
		if err != nil {
			return nil, err
		}
		if end.Before(start) {
			return nil, fmt.Errorf("end date is before start date")
		}
		var days []time.Time
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			days = append(days, d)
		}
		return days, nil
	}

	// single date
	d, err := parseDate(args[0])
	if err != nil {
		return nil, err
	}
	return []time.Time{d}, nil
}

func parseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return t, nil
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// listAllDays returns all recorded days sorted ascending.
func listAllDays(store *storage.Storage) ([]time.Time, error) {
	days, err := store.ListDays()
	if err != nil {
		return nil, err
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days, nil
}
