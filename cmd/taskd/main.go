package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/update"
)

func main() {
	program := tea.NewProgram(update.NewModel())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "taskd failed: %v\n", err)
		os.Exit(1)
	}
}
