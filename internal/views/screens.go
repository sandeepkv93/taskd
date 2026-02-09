package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type InboxPanelData struct {
	QuickAddView string
	ListView     string
}

type TodayItemData struct {
	ID          string
	Title       string
	Bucket      string
	ScheduledAt string
	DueAt       string
	Priority    string
}

type TodayPanelData struct {
	ListView   string
	Items      []TodayItemData
	SelectedID string
	Collapsed  map[string]bool
}

type CalendarAgendaItemData struct {
	ID    string
	Title string
	Date  string
	Time  string
	Kind  string
}

type CalendarPanelData struct {
	Mode      string
	FocusDate string
	TableView string
	Items     []CalendarAgendaItemData
	Selected  *CalendarAgendaItemData
}

type FocusPanelData struct {
	TaskTitle          string
	Phase              string
	Timer              string
	ProgressView       string
	ProgressPct        int
	CompletedPomodoros int
	ShowEndPrompt      bool
}

type HelpPanelData struct {
	CurrentView string
	Bindings    []string
	HelpView    string
}

type SuggestionData struct {
	Title   string
	Energy  string
	Minutes int
	Reason  string
}

type ProductivityPanelData struct {
	TemporalDebtScore int
	TemporalDebtLabel string
	Suggestions       []SuggestionData
}

type TodayMetadataData struct {
	SelectedID       string
	Priority         string
	Tags             []string
	NotesEditorView  string
	MarkdownMetaView string
}

type RecurrenceEditorData struct {
	Active       bool
	RuleType     string
	IntervalText string
	ErrorText    string
	Preview      []string
}

var (
	sectionHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	cardStyle          = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).MarginRight(1)
	subtleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	focusTimerStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	accentStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
)

func RenderInboxPanel(data InboxPanelData) string {
	var b strings.Builder
	b.WriteString(accentStyle.Render("inbox:") + "\n")
	b.WriteString(data.QuickAddView + "\n")
	b.WriteString("actions: [enter]add [space]select [x]all [u]clear [s]schedule [g]tag\n")
	b.WriteString(data.ListView)
	return strings.TrimSpace(b.String())
}

func RenderTodayPanel(data TodayPanelData) string {
	scheduled := make([]TodayItemData, 0)
	anytime := make([]TodayItemData, 0)
	overdue := make([]TodayItemData, 0)
	for _, item := range data.Items {
		switch item.Bucket {
		case "Scheduled":
			scheduled = append(scheduled, item)
		case "Overdue":
			overdue = append(overdue, item)
		default:
			anytime = append(anytime, item)
		}
	}

	var b strings.Builder
	b.WriteString(accentStyle.Render("today:") + "\n")
	b.WriteString("actions: [j/k]move [z]collapse [1]today [2]inbox [3]calendar [4]focus\n")
	b.WriteString(data.ListView + "\n")

	scheduledBlock := renderTodaySection("Scheduled", scheduled, data.SelectedID, data.Collapsed["Scheduled"])
	anytimeBlock := renderTodaySection("Anytime", anytime, data.SelectedID, data.Collapsed["Anytime"])
	overdueBlock := renderTodaySection("Overdue", overdue, data.SelectedID, data.Collapsed["Overdue"])

	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		cardStyle.Width(20).Render(scheduledBlock),
		cardStyle.Width(20).Render(anytimeBlock),
	)
	b.WriteString(topRow + "\n")
	b.WriteString(cardStyle.Width(42).Render(overdueBlock))
	return strings.TrimSpace(b.String())
}

func RenderCalendarPanel(data CalendarPanelData) string {
	var b strings.Builder
	b.WriteString(accentStyle.Render("calendar:") + "\n")
	b.WriteString(fmt.Sprintf("mode: %s | focus: %s\n", data.Mode, data.FocusDate))
	b.WriteString("actions: [d]day [w]week [m]month [h/l]period [j/k]agenda\n")
	b.WriteString(cardStyle.Width(42).Render(data.TableView) + "\n")

	grouped := make(map[string][]CalendarAgendaItemData)
	keys := make([]string, 0)
	for _, item := range data.Items {
		if _, ok := grouped[item.Date]; !ok {
			keys = append(keys, item.Date)
		}
		grouped[item.Date] = append(grouped[item.Date], item)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		b.WriteString("(agenda empty)")
		return b.String()
	}

	var agenda strings.Builder
	for _, day := range keys {
		agenda.WriteString(fmt.Sprintf("\n%s:\n", day))
		items := grouped[day]
		sort.SliceStable(items, func(i, j int) bool { return items[i].Time < items[j].Time })
		for _, item := range items {
			cursor := " "
			if data.Selected != nil && data.Selected.ID == item.ID {
				cursor = ">"
			}
			agenda.WriteString(fmt.Sprintf("%s [%s] %s %s\n", cursor, strings.ToUpper(item.Kind), item.Time, item.Title))
		}
	}

	rightMeta := "agenda-metadata:\n(no selection)"
	if data.Selected != nil {
		rightMeta = strings.TrimSpace(fmt.Sprintf("agenda-metadata:\nid: %s\nkind: %s\nwhen: %s %s\n",
			data.Selected.ID,
			data.Selected.Kind,
			data.Selected.Date,
			data.Selected.Time,
		))
	}

	b.WriteString(cardStyle.Width(42).Render(strings.TrimSpace(agenda.String())) + "\n")
	b.WriteString(cardStyle.Width(42).Render(rightMeta))
	return strings.TrimSpace(b.String())
}

func RenderFocusPanel(data FocusPanelData) string {
	var b strings.Builder
	b.WriteString(accentStyle.Render("focus:") + "\n")
	if data.TaskTitle != "" {
		b.WriteString(fmt.Sprintf("task: %s\n", data.TaskTitle))
	} else {
		b.WriteString("task: (none selected)\n")
	}
	timerCard := cardStyle.Width(42).Render(strings.TrimSpace(fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		sectionHeaderStyle.Render("phase: "+strings.ToUpper(data.Phase)),
		focusTimerStyle.Render("timer: "+data.Timer),
		"progress: "+data.ProgressView+fmt.Sprintf(" %d%%", data.ProgressPct),
		subtleStyle.Render(fmt.Sprintf("pomodoros completed: %d", data.CompletedPomodoros)),
	)))
	b.WriteString(timerCard + "\n")
	b.WriteString("actions: [space]start/pause [r]reset [n]next-phase\n")
	if data.ShowEndPrompt {
		b.WriteString("prompt: session ended, press [n] to continue")
	}
	return strings.TrimSpace(b.String())
}

func RenderCommandPalette(active bool, input string) string {
	if !active {
		return ""
	}
	return cardStyle.Width(42).Render(fmt.Sprintf("command palette\n/%s", input))
}

func RenderNotification(level string, body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return fmt.Sprintf("\nnotification: [%s] %s", strings.ToUpper(level), body)
}

func RenderProductivityPanel(data ProductivityPanelData) string {
	if data.TemporalDebtScore == 0 && len(data.Suggestions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nproductivity:\n")
	b.WriteString(fmt.Sprintf("temporal-debt: %d (%s)\n", data.TemporalDebtScore, data.TemporalDebtLabel))
	if len(data.Suggestions) > 0 {
		b.WriteString("suggestions:\n")
		for i, sg := range data.Suggestions {
			if i >= 2 {
				b.WriteString(fmt.Sprintf("- +%d more…\n", len(data.Suggestions)-i))
				break
			}
			b.WriteString(fmt.Sprintf("- %s [%s, %dm]\n", sg.Title, sg.Energy, sg.Minutes))
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func RenderHelpPanel(data HelpPanelData) string {
	left := cardStyle.Width(42).Render(fmt.Sprintf("help:\nglobal:\n%s view:\n%s",
		strings.ToLower(data.CurrentView),
		strings.Join(data.Bindings, "\n"),
	))
	right := cardStyle.Width(42).Render(data.HelpView)
	return strings.TrimSpace(left + "\n" + right)
}

func RenderTodayMetadataPane(data TodayMetadataData) string {
	if strings.TrimSpace(data.SelectedID) == "" {
		return "metadata:\n(no selection)"
	}
	tags := strings.Join(data.Tags, ",")
	return fmt.Sprintf("metadata:\nid: %s\npriority: %s\ntags: %s\n\nnotes-editor:\n%s\n\nmarkdown-preview:\n%s",
		data.SelectedID,
		data.Priority,
		tags,
		data.NotesEditorView,
		data.MarkdownMetaView,
	)
}

func RenderRecurrenceEditor(data RecurrenceEditorData) string {
	if !data.Active {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nrecurrence-editor:\n")
	b.WriteString("keys: [tab] field [enter] preview [esc] close\n")
	b.WriteString(fmt.Sprintf("type: %s\n", data.RuleType))
	b.WriteString(fmt.Sprintf("interval: %s\n", data.IntervalText))
	if data.ErrorText != "" {
		b.WriteString("error: " + data.ErrorText + "\n")
	}
	if len(data.Preview) > 0 {
		b.WriteString("preview:\n")
		for _, item := range data.Preview {
			b.WriteString("- " + item + "\n")
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func renderTodaySection(title string, items []TodayItemData, selectedID string, collapsed bool) string {
	var b strings.Builder
	marker := "[-]"
	if collapsed {
		marker = "[+]"
	}
	b.WriteString(sectionHeaderStyle.Render(fmt.Sprintf("%s %s:", marker, title)))
	b.WriteString("\n")
	if collapsed {
		b.WriteString("  (collapsed)\n")
		return strings.TrimSpace(b.String())
	}
	if len(items) == 0 {
		b.WriteString("  (none)\n")
		return strings.TrimSpace(b.String())
	}
	for _, item := range items {
		cursor := " "
		if selectedID == item.ID {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf("%s %s %s", cursor, urgencyBadge(item), item.Title))
		if item.ScheduledAt != "" {
			b.WriteString(fmt.Sprintf(" @%s", item.ScheduledAt))
		}
		if item.DueAt != "" {
			b.WriteString(fmt.Sprintf(" due:%s", item.DueAt))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func urgencyBadge(item TodayItemData) string {
	if item.Bucket == "Overdue" || item.Priority == "Critical" {
		return "[RED]"
	}
	if item.Bucket == "Scheduled" || item.Priority == "High" {
		return "[YELLOW]"
	}
	return "[GREEN]"
}
