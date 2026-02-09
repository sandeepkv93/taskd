package update

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleFocusKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case " ":
		if m.Focus.Running {
			m.Focus.Running = false
			m.Status = StatusBar{Text: "focus paused", IsError: false}
			return m, nil
		}
		if m.Focus.RemainingSec <= 0 {
			m.Focus.RemainingSec = m.currentFocusTotal()
		}
		m.Focus.Running = true
		m.Status = StatusBar{Text: "focus running", IsError: false}
		return m, focusTickCmd()
	case "r":
		m.Focus.Running = false
		m.Focus.RemainingSec = m.currentFocusTotal()
		m.Status = StatusBar{Text: "focus reset", IsError: false}
		return m, nil
	case "n":
		m.completeFocusPhase()
		return m, nil
	}
	return m, nil
}

func (m Model) onFocusTick() (tea.Model, tea.Cmd) {
	if !m.Focus.Running {
		return m, nil
	}
	if m.Focus.RemainingSec > 0 {
		m.Focus.RemainingSec--
	}
	if m.Focus.RemainingSec == 0 {
		m.Focus.Running = false
		if m.Focus.Phase == FocusPhaseWork {
			m.Status = StatusBar{Text: "work session complete; press n to start break", IsError: false}
		} else {
			m.Status = StatusBar{Text: "break complete; press n for next focus block", IsError: false}
		}
		return m, nil
	}
	return m, focusTickCmd()
}

func (m *Model) bootstrapFocusTask() {
	if m.Focus.TaskID != "" {
		return
	}
	m.Focus.TaskID = m.SelectedTaskID
	if item, ok := m.currentTodayItem(); ok {
		m.Focus.TaskID = item.ID
		m.Focus.TaskTitle = item.Title
		return
	}
	if m.Focus.TaskID != "" {
		m.Focus.TaskTitle = m.Focus.TaskID
	}
}

func (m *Model) completeFocusPhase() {
	if m.Focus.Phase == FocusPhaseWork {
		m.Focus.CompletedPomodoros++
		if m.Focus.TaskID != "" {
			m.CompletedTasks[m.Focus.TaskID] = true
			if err := m.persistCompletedTaskState(); err != nil {
				m.Status = StatusBar{Text: fmt.Sprintf("persist completion state failed: %v", err), IsError: true}
				return
			}
		}
		m.Focus.Phase = FocusPhaseBreak
		m.Focus.RemainingSec = m.Focus.BreakDurationSec
		m.Focus.Running = false
		m.Status = StatusBar{Text: "break ready", IsError: false}
		return
	}
	m.Focus.Phase = FocusPhaseWork
	m.Focus.RemainingSec = m.Focus.WorkDurationSec
	m.Focus.Running = false
	m.Status = StatusBar{Text: "focus block ready", IsError: false}
}

func (m Model) currentFocusTotal() int {
	if m.Focus.Phase == FocusPhaseBreak {
		return m.Focus.BreakDurationSec
	}
	return m.Focus.WorkDurationSec
}

func focusTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return FocusTickMsg{} })
}
