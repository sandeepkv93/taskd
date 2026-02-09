package views

import (
	"fmt"
	"sort"
	"strings"
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

func RenderInboxPanel(data InboxPanelData) string {
	var b strings.Builder
	b.WriteString("inbox:\n")
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
	b.WriteString("today:\n")
	b.WriteString("actions: [j/k]move [1]today [2]inbox [3]calendar [4]focus\n")
	b.WriteString(data.ListView + "\n")
	renderTodaySection(&b, "Scheduled", scheduled, data.SelectedID)
	renderTodaySection(&b, "Anytime", anytime, data.SelectedID)
	renderTodaySection(&b, "Overdue", overdue, data.SelectedID)
	return strings.TrimSpace(b.String())
}

func RenderCalendarPanel(data CalendarPanelData) string {
	var b strings.Builder
	b.WriteString("calendar:\n")
	b.WriteString(fmt.Sprintf("mode: %s | focus: %s\n", data.Mode, data.FocusDate))
	b.WriteString("actions: [d]day [w]week [m]month [h/l]period [j/k]agenda\n")
	b.WriteString(data.TableView + "\n")

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

	for _, day := range keys {
		b.WriteString(fmt.Sprintf("\n%s:\n", day))
		items := grouped[day]
		sort.SliceStable(items, func(i, j int) bool { return items[i].Time < items[j].Time })
		for _, item := range items {
			cursor := " "
			if data.Selected != nil && data.Selected.ID == item.ID {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s [%s] %s %s\n", cursor, strings.ToUpper(item.Kind), item.Time, item.Title))
		}
	}

	if data.Selected != nil {
		b.WriteString("\nagenda-metadata:\n")
		b.WriteString(fmt.Sprintf("id: %s\n", data.Selected.ID))
		b.WriteString(fmt.Sprintf("kind: %s\n", data.Selected.Kind))
		b.WriteString(fmt.Sprintf("when: %s %s\n", data.Selected.Date, data.Selected.Time))
	}
	return strings.TrimSpace(b.String())
}

func RenderFocusPanel(data FocusPanelData) string {
	var b strings.Builder
	b.WriteString("focus:\n")
	if data.TaskTitle != "" {
		b.WriteString(fmt.Sprintf("task: %s\n", data.TaskTitle))
	} else {
		b.WriteString("task: (none selected)\n")
	}
	b.WriteString(fmt.Sprintf("phase: %s\n", strings.ToUpper(data.Phase)))
	b.WriteString(fmt.Sprintf("timer: %s\n", data.Timer))
	b.WriteString(fmt.Sprintf("progress: %s %d%%\n", data.ProgressView, data.ProgressPct))
	b.WriteString(fmt.Sprintf("pomodoros completed: %d\n", data.CompletedPomodoros))
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
	return fmt.Sprintf("command: /%s", input)
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
		for _, sg := range data.Suggestions {
			b.WriteString(fmt.Sprintf("- %s [%s, %dm] %s\n", sg.Title, sg.Energy, sg.Minutes, sg.Reason))
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func RenderHelpPanel(data HelpPanelData) string {
	return fmt.Sprintf("help:\nglobal:\n%s view:\n%s\n%s",
		strings.ToLower(data.CurrentView),
		strings.Join(data.Bindings, "\n"),
		data.HelpView,
	)
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

func renderTodaySection(b *strings.Builder, title string, items []TodayItemData, selectedID string) {
	b.WriteString(fmt.Sprintf("\n%s:\n", title))
	if len(items) == 0 {
		b.WriteString("  (none)\n")
		return
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
