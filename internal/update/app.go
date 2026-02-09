package update

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/scheduler"
	"github.com/sandeepkv93/taskd/internal/views"
)

func (m *Model) initBubbleComponents() {
	m.inboxList = list.New([]list.Item{}, list.NewDefaultDelegate(), 56, 12)
	m.inboxList.Title = "Inbox (list)"
	m.inboxList.SetShowHelp(false)
	m.inboxList.SetFilteringEnabled(false)

	m.todayList = list.New([]list.Item{}, list.NewDefaultDelegate(), 56, 12)
	m.todayList.Title = "Today (list)"
	m.todayList.SetShowHelp(false)
	m.todayList.SetFilteringEnabled(false)

	cols := []table.Column{
		{Title: "Date", Width: 12},
		{Title: "Time", Width: 7},
		{Title: "Kind", Width: 8},
		{Title: "Title", Width: 24},
	}
	m.calendarTable = table.New(table.WithColumns(cols), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(10))

	m.quickAddInput = textinput.New()
	m.quickAddInput.Prompt = "add> "
	m.quickAddInput.CharLimit = 256
	m.quickAddInput.Width = 42

	m.commandInput = textinput.New()
	m.commandInput.Prompt = "/"
	m.commandInput.CharLimit = 256
	m.commandInput.Width = 48

	m.notesArea = textarea.New()
	m.notesArea.SetWidth(54)
	m.notesArea.SetHeight(8)
	m.notesArea.ShowLineNumbers = false
	m.notesArea.Placeholder = "Task notes (markdown)"

	m.focusProgress = progress.New(progress.WithDefaultGradient())

	m.syncSpinner = spinner.New()
	m.syncSpinner.Spinner = spinner.Dot

	m.helpModel = help.New()
	m.metaViewport = viewport.New(54, 12)
}

func (m *Model) syncBubbleData() {
	listWidth, listHeight, tableHeight, notesHeight, viewportHeight := densityDimensions(m.uiDensity)
	m.inboxList.SetSize(listWidth, listHeight)
	m.todayList.SetSize(listWidth, listHeight)
	m.calendarTable.SetHeight(tableHeight)
	m.notesArea.SetHeight(notesHeight)
	m.metaViewport.Height = viewportHeight

	inboxItems := make([]list.Item, 0, len(m.Inbox.Items))
	for _, item := range m.Inbox.Items {
		desc := item.ScheduledFor
		if desc == "" {
			desc = strings.Join(item.Tags, ",")
		}
		inboxItems = append(inboxItems, listItem{title: item.Title, description: desc})
	}
	m.inboxList.SetItems(inboxItems)
	if len(inboxItems) > 0 {
		m.inboxList.Select(m.Inbox.Cursor)
	}

	todayItems := make([]list.Item, 0, len(m.Today.Items))
	for _, item := range m.Today.Items {
		desc := fmt.Sprintf("%s | %s", item.Bucket, item.Priority)
		todayItems = append(todayItems, listItem{title: item.Title, description: desc})
	}
	m.todayList.SetItems(todayItems)
	if len(todayItems) > 0 {
		m.todayList.Select(m.Today.Cursor)
	}

	rows := make([]table.Row, 0, len(m.Calendar.Items))
	for _, item := range m.Calendar.Items {
		rows = append(rows, table.Row{item.Date, item.Time, strings.ToUpper(item.Kind), item.Title})
	}
	m.calendarTable.SetRows(rows)
	if len(rows) > 0 && m.Calendar.Cursor < len(rows) {
		m.calendarTable.SetCursor(m.Calendar.Cursor)
	}

	m.quickAddInput.SetValue(m.Inbox.Input)
	m.commandInput.SetValue(m.Palette.Input)
	if m.CurrentView == ViewInbox && m.Inbox.CaptureMode {
		m.quickAddInput.Focus()
	} else {
		m.quickAddInput.Blur()
	}
	if m.Palette.Active {
		m.commandInput.Focus()
	}

	if sel, ok := m.currentTodayItem(); ok {
		md := sel.Notes
		if strings.TrimSpace(md) == "" {
			md = "_No notes_"
		}
		m.notesArea.SetValue(md)
		m.metaViewport.SetContent(views.RenderMarkdown(md))
	}

	total := m.currentFocusTotal()
	pct := 0.0
	if total > 0 {
		pct = float64(total-m.Focus.RemainingSec) / float64(total)
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	_ = m.focusProgress.SetPercent(pct)
}

func densityDimensions(level int) (listWidth int, listHeight int, tableHeight int, notesHeight int, viewportHeight int) {
	switch level {
	case 2:
		return 60, 14, 12, 10, 14
	case 3:
		return 64, 16, 14, 12, 16
	default:
		return 56, 12, 10, 8, 12
	}
}

func (m *Model) ensureInboxState() {
	if m.Inbox.Selected == nil {
		m.Inbox.Selected = make(map[string]bool)
	}
	if m.Inbox.NextID <= 0 {
		m.Inbox.NextID = 1
	}
}

func (m *Model) ensureTodayState() {
	if m.todayCollapsed == nil {
		m.todayCollapsed = map[TodayBucket]bool{
			TodayBucketScheduled: false,
			TodayBucketAnytime:   false,
			TodayBucketOverdue:   false,
		}
	}
	if m.Today.Cursor < 0 {
		m.Today.Cursor = 0
	}
	if m.Today.Cursor >= len(m.Today.Items) && len(m.Today.Items) > 0 {
		m.Today.Cursor = len(m.Today.Items) - 1
	}
	if len(m.Today.Items) > 0 && m.SelectedTaskID == "" {
		m.syncSelectedTaskToTodayCursor()
	}
}

func (m *Model) toggleTodaySectionCollapse() {
	selected, ok := m.currentTodayItem()
	if !ok {
		return
	}
	bucket := selected.Bucket
	m.todayCollapsed[bucket] = !m.todayCollapsed[bucket]
	state := "expanded"
	if m.todayCollapsed[bucket] {
		state = "collapsed"
	}
	m.Status = StatusBar{
		Text:    fmt.Sprintf("%s section %s", strings.ToLower(string(bucket)), state),
		IsError: false,
	}
}

func (m *Model) cycleDensity() {
	m.uiDensity++
	if m.uiDensity > 3 {
		m.uiDensity = 1
	}
	m.Status = StatusBar{
		Text:    fmt.Sprintf("density level: %d", m.uiDensity),
		IsError: false,
	}
}

func (m Model) renderInboxView() string {
	return views.RenderInboxPanel(views.InboxPanelData{
		QuickAddView: m.quickAddInput.View(),
		ListView:     m.inboxList.View(),
		CaptureMode:  m.Inbox.CaptureMode,
	})
}

func (m Model) renderTodayView() string {
	items := make([]views.TodayItemData, 0, len(m.Today.Items))
	for _, item := range m.Today.Items {
		items = append(items, views.TodayItemData{
			ID:          item.ID,
			Title:       item.Title,
			Bucket:      string(item.Bucket),
			ScheduledAt: item.ScheduledAt,
			DueAt:       item.DueAt,
			Priority:    item.Priority,
		})
	}
	return views.RenderTodayPanel(views.TodayPanelData{
		ListView:   m.todayList.View(),
		Items:      items,
		SelectedID: m.SelectedTaskID,
		Collapsed: map[string]bool{
			string(TodayBucketScheduled): m.todayCollapsed[TodayBucketScheduled],
			string(TodayBucketAnytime):   m.todayCollapsed[TodayBucketAnytime],
			string(TodayBucketOverdue):   m.todayCollapsed[TodayBucketOverdue],
		},
	})
}

func (m Model) renderCalendarView() string {
	items := make([]views.CalendarAgendaItemData, 0, len(m.Calendar.Items))
	for _, item := range m.Calendar.Items {
		items = append(items, views.CalendarAgendaItemData{
			ID:    item.ID,
			Title: item.Title,
			Date:  item.Date,
			Time:  item.Time,
			Kind:  item.Kind,
		})
	}
	var selected *views.CalendarAgendaItemData
	if s, ok := m.currentAgendaItem(); ok {
		selected = &views.CalendarAgendaItemData{
			ID:    s.ID,
			Title: s.Title,
			Date:  s.Date,
			Time:  s.Time,
			Kind:  s.Kind,
		}
	}
	return views.RenderCalendarPanel(views.CalendarPanelData{
		Mode:      string(m.Calendar.Mode),
		FocusDate: m.Calendar.FocusDate.Format("2006-01-02"),
		TableView: m.calendarTable.View(),
		Items:     items,
		Selected:  selected,
	})
}

func (m Model) renderFocusView() string {
	total := m.currentFocusTotal()
	progress := 0.0
	if total > 0 {
		progress = float64(total-m.Focus.RemainingSec) / float64(total)
	}

	return views.RenderFocusPanel(views.FocusPanelData{
		TaskTitle:          m.Focus.TaskTitle,
		Phase:              string(m.Focus.Phase),
		Timer:              formatDuration(m.Focus.RemainingSec),
		ProgressView:       m.focusProgress.ViewAs(progress),
		ProgressPct:        int(progress * 100),
		CompletedPomodoros: m.Focus.CompletedPomodoros,
		ShowEndPrompt:      m.Focus.RemainingSec == 0,
	})
}

func waitForReminderCmd(ch <-chan scheduler.ReminderEvent) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return ReminderDueMsg{Event: ev}
	}
}
