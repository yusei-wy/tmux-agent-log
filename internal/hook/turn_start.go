package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/git"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

type turnStartInput struct {
	SessionID           string `json:"session_id"`
	Cwd                 string `json:"cwd"`
	Prompt              string `json:"prompt"`
	TranscriptMessageID string `json:"transcript_message_id"`
}

func RunTurnStart(stdin io.Reader) error {
	var in turnStartInput
	if err := json.NewDecoder(stdin).Decode(&in); err != nil {
		return fmt.Errorf("decode turn_start input: %w", err)
	}

	if in.SessionID == "" || in.Cwd == "" {
		return nil
	}

	sDir, err := config.SessionDir(in.Cwd, in.SessionID)
	if err != nil {
		return fmt.Errorf("resolve session dir: %w", err)
	}

	meta, err := storage.ReadSessionMeta(sDir)
	if err != nil {
		return fmt.Errorf("read session meta: %w", err)
	}

	headSHA := ""
	if meta.GitTracked {
		if sha, err := git.HeadSHA(in.Cwd); err == nil {
			headSHA = sha
		}
	}

	turn := storage.TurnOpen{
		ID:                  "turn-" + uuid.NewString(),
		StartedAt:           time.Now().UTC(),
		UserPromptPreview:   promptPreview(in.Prompt, 2, 400),
		HeadSHAPre:          headSHA,
		TranscriptMessageID: in.TranscriptMessageID,
	}

	return storage.AppendTurnOpen(filepath.Join(sDir, "turns.jsonl"), turn)
}
