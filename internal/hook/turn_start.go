package hook

import (
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
	if err := ReadInput(stdin, &in); err != nil {
		return err
	}
	if in.SessionID == "" || in.Cwd == "" {
		return nil
	}

	sDir, err := config.SessionDir(in.Cwd, in.SessionID)
	if err != nil {
		return err
	}

	meta, err := storage.ReadSessionMeta(sDir)
	if err != nil {
		return err
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
		UserPromptPreview:   previewFirstLines(in.Prompt, 400),
		HeadSHAPre:          headSHA,
		TranscriptMessageID: in.TranscriptMessageID,
	}

	return storage.AppendTurnOpen(filepath.Join(sDir, "turns.jsonl"), turn)
}

