package update

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/sandeepkv93/taskd/internal/views"
)

type KeyBinding struct {
	Key    string
	Action string
}

type helpKeyMap struct {
	short []key.Binding
	full  [][]key.Binding
}

func (k helpKeyMap) ShortHelp() []key.Binding  { return k.short }
func (k helpKeyMap) FullHelp() [][]key.Binding { return k.full }

func (m Model) renderHelpIfVisible() string {
	if !m.HelpVisible {
		return ""
	}
	return m.renderHelpView()
}

func (m Model) renderHelpView() string {
	bindings := m.helpBindings()
	var plain []string
	for _, kb := range m.viewBindings() {
		plain = append(plain, fmt.Sprintf("- %s: %s", kb.Key, kb.Action))
	}
	return views.RenderHelpPanel(views.HelpPanelData{
		CurrentView: string(m.CurrentView),
		Bindings:    plain,
		HelpView: m.helpModel.View(helpKeyMap{
			short: bindings,
			full:  [][]key.Binding{bindings},
		}),
	})
}

func (m Model) globalBindings() []KeyBinding {
	return []KeyBinding{
		{Key: m.Keys.Today, Action: "switch to Today"},
		{Key: m.Keys.Inbox, Action: "switch to Inbox"},
		{Key: m.Keys.Calendar, Action: "switch to Calendar"},
		{Key: m.Keys.Focus, Action: "switch to Focus"},
		{Key: "/", Action: "open command palette"},
		{Key: "D", Action: "cycle density"},
		{Key: m.Keys.Help, Action: "toggle help panel"},
		{Key: m.Keys.Quit, Action: "quit app"},
	}
}

func (m Model) viewBindings() []KeyBinding {
	switch m.CurrentView {
	case ViewInbox:
		return []KeyBinding{
			{Key: "enter", Action: "capture inbox item"},
			{Key: "j/k", Action: "move cursor"},
			{Key: "space", Action: "toggle select"},
			{Key: "x/u", Action: "select all / clear selection"},
			{Key: "s/g", Action: "bulk schedule / bulk tag"},
		}
	case ViewToday:
		return []KeyBinding{
			{Key: "j/k", Action: "move selection"},
			{Key: "z", Action: "collapse/expand selected section"},
		}
	case ViewCalendar:
		return []KeyBinding{
			{Key: "d/w/m", Action: "day/week/month mode"},
			{Key: "h/l", Action: "previous/next period"},
			{Key: "j/k", Action: "move agenda cursor"},
		}
	case ViewFocus:
		return []KeyBinding{
			{Key: "space", Action: "start/pause timer"},
			{Key: "r", Action: "reset timer"},
			{Key: "n", Action: "next focus phase"},
		}
	default:
		return []KeyBinding{{Key: "-", Action: "no contextual bindings"}}
	}
}

func (m Model) helpBindings() []key.Binding {
	out := make([]key.Binding, 0, len(m.globalBindings())+len(m.viewBindings()))
	for _, kb := range m.globalBindings() {
		out = append(out, key.NewBinding(key.WithKeys(kb.Key), key.WithHelp(kb.Key, kb.Action)))
	}
	for _, kb := range m.viewBindings() {
		out = append(out, key.NewBinding(key.WithKeys(kb.Key), key.WithHelp(kb.Key, kb.Action)))
	}
	return out
}
