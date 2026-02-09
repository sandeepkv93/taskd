package views

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type AppData struct {
	Header       string
	LeftPane     string
	RightPane    string
	StatusLine   string
	Footer       string
	Notification string
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func RenderApp(data AppData) string {
	left := panelStyle.Width(58).Render(data.LeftPane)
	right := panelStyle.Width(58).Render(data.RightPane)
	row := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	status := statusStyle.Render(data.StatusLine)
	if strings.Contains(strings.ToLower(data.StatusLine), "error") {
		status = errorStyle.Render(data.StatusLine)
	}

	lines := []string{
		headerStyle.Render(data.Header),
		row,
		status,
	}
	if data.Notification != "" {
		lines = append(lines, panelStyle.Render(data.Notification))
	}
	if data.Footer != "" {
		lines = append(lines, footerStyle.Render(data.Footer))
	}
	return strings.Join(lines, "\n")
}

func RenderMarkdown(md string) string {
	if strings.TrimSpace(md) == "" {
		return ""
	}
	out, err := glamour.Render(md, "dark")
	if err != nil {
		return md
	}
	return strings.TrimSpace(out)
}
