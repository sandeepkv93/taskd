package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type completionState struct {
	CompletedTaskIDs []string `json:"completed_task_ids"`
}

func (m *Model) persistCompletedTaskState() error {
	if strings.TrimSpace(m.stateFilePath) == "" {
		return nil
	}
	dir := filepath.Dir(m.stateFilePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	ids := make([]string, 0, len(m.CompletedTasks))
	for id, done := range m.CompletedTasks {
		if done && strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	payload, err := json.MarshalIndent(completionState{CompletedTaskIDs: ids}, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.stateFilePath + ".tmp"
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.stateFilePath)
}

func loadCompletedTaskState(path string) (map[string]bool, error) {
	out := make(map[string]bool)
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return out, nil
	}
	raw, err := os.ReadFile(trimmed)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return out, nil
	}
	var state completionState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	for _, id := range state.CompletedTaskIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out[id] = true
	}
	return out, nil
}
