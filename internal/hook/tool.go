package hook

import (
	"encoding/json"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

type toolHookInput struct {
	SessionID    string          `json:"session_id"`
	Cwd          string          `json:"cwd"`
	TurnID       string          `json:"turn_id"`
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolResponse struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	} `json:"tool_response"`
}

func RunToolPre(stdin io.Reader) error {
	return runToolHook(stdin, storage.EventPhasePre)
}

func RunToolPost(stdin io.Reader) error {
	return runToolHook(stdin, storage.EventPhasePost)
}

func runToolHook(stdin io.Reader, phase storage.EventPhase) error {
	var in toolHookInput
	if err := json.NewDecoder(stdin).Decode(&in); err != nil {
		return err
	}
	if in.SessionID == "" || in.Cwd == "" {
		return nil
	}

	sDir, err := config.SessionDir(in.Cwd, in.SessionID)
	if err != nil {
		return err
	}

	turnID := in.TurnID
	if turnID == "" {
		latest, err := storage.LatestOpenTurnID(filepath.Join(sDir, "turns.jsonl"))
		if err != nil {
			return err
		}
		turnID = latest
	}
	if turnID == "" {
		return nil
	}

	event := storage.Event{
		ID:          "evt-" + uuid.NewString(),
		TurnID:      turnID,
		Ts:          time.Now().UTC(),
		Tool:        in.ToolName,
		ArgsPreview: truncate(string(in.ToolInput), 200),
		Phase:       phase,
	}
	if phase == storage.EventPhasePost {
		event.Success = in.ToolResponse.Success
		event.ErrorMessage = in.ToolResponse.Error
	}

	return storage.AppendEvent(filepath.Join(sDir, "events.jsonl"), event)
}
