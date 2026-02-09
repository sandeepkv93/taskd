package update

import (
	"fmt"
	"strings"
)

func (m *Model) refreshProductivitySignals() {
	score := m.computeTemporalDebtScore()
	m.Productivity.Signals.TemporalDebtScore = score
	switch {
	case score >= 7:
		m.Productivity.Signals.TemporalDebtLabel = "high"
	case score >= 4:
		m.Productivity.Signals.TemporalDebtLabel = "medium"
	default:
		m.Productivity.Signals.TemporalDebtLabel = "low"
	}
	m.Productivity.Signals.Suggestions = m.computeEnergySuggestions(m.Productivity.AvailableMinutes, 3)
}

func (m Model) computeTemporalDebtScore() int {
	score := 0
	for _, item := range m.Today.Items {
		if item.Bucket == TodayBucketOverdue {
			score += 2
		}
		notes := strings.ToLower(item.Notes)
		if strings.Contains(notes, "snoozed") {
			score++
		}
		if item.Priority == "Critical" && item.Bucket == TodayBucketOverdue {
			score++
		}
	}
	if score > 10 {
		return 10
	}
	return score
}

func (m Model) computeEnergySuggestions(availableMinutes int, limit int) []Suggestion {
	if availableMinutes <= 0 || limit <= 0 {
		return nil
	}
	out := make([]Suggestion, 0, limit)
	for _, item := range m.Today.Items {
		energy := inferEnergyFromTodayItem(item)
		minutes := estimateMinutesForEnergy(energy)
		if minutes > availableMinutes {
			continue
		}
		reason := fmt.Sprintf("fits %d-minute window", availableMinutes)
		if item.Bucket == TodayBucketOverdue {
			reason = "overdue and still feasible now"
		}
		out = append(out, Suggestion{
			TaskID:  item.ID,
			Title:   item.Title,
			Reason:  reason,
			Energy:  energy,
			Minutes: minutes,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func inferEnergyFromTodayItem(item TodayItem) string {
	combined := strings.ToLower(item.Title + " " + item.Notes + " " + strings.Join(item.Tags, " "))
	switch {
	case strings.Contains(combined, "call"), strings.Contains(combined, "meeting"), strings.Contains(combined, "standup"):
		return "Social"
	case strings.Contains(combined, "review"), strings.Contains(combined, "docs"), strings.Contains(combined, "write"):
		return "Light"
	case strings.Contains(combined, "tax"), strings.Contains(combined, "budget"), strings.Contains(combined, "email"):
		return "Low"
	default:
		return "Deep"
	}
}

func estimateMinutesForEnergy(energy string) int {
	switch strings.ToLower(energy) {
	case "deep":
		return 90
	case "social":
		return 45
	case "light":
		return 30
	default:
		return 20
	}
}
