package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/scheduler"
	"github.com/sandeepkv93/taskd/internal/update"
)

func main() {
	reminderEngine := scheduler.NewEngine(64)
	reminderEngine.Start()
	defer reminderEngine.Stop()

	program := tea.NewProgram(update.NewModelWithRuntime(
		reminderEngine,
		update.DesktopNotificationsEnabledFromEnv(),
		update.ExecDesktopNotifier{},
	))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "taskd failed: %v\n", err)
		os.Exit(1)
	}
}
