package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/kubeone/go-time-tracker/internal/storage"
	"github.com/kubeone/go-time-tracker/internal/tracker"
	"github.com/kubeone/go-time-tracker/internal/ui"
)

func dataDir() string {
	if runtime.GOOS == "windows" {
		if dir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(dir, "gtt")
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".time-tracker")
}

func NewRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:     "gtt",
		Short:   "Terminal time tracker",
		Long:    "A terminal-based time tracker that stores entries as Markdown files.",
		Version: version,
		RunE:    runTUI,
	}

	root.AddCommand(newReportCmd())
	return root
}

func runTUI(cmd *cobra.Command, args []string) error {
	store, err := storage.New(dataDir())
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	cfg, err := store.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	today := time.Now()
	tr, err := tracker.New(store, today)
	if err != nil {
		return fmt.Errorf("init tracker: %w", err)
	}

	app := ui.NewApp(tr, store, cfg.Profiles)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err = p.Run()

	return err
}
