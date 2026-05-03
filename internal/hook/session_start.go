package hook

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/git"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

type sessionStartInput struct {
	SessionID      string `json:"session_id"`
	Cwd            string `json:"cwd"`
	TranscriptPath string `json:"transcript_path"`
}

func RunSessionStart(stdin io.Reader) error {
	var in sessionStartInput
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

	meta := storage.SessionMeta{
		ClaudeSessionID: in.SessionID,
		TmuxPane:        os.Getenv("TMUX_PANE"),
		Cwd:             in.Cwd,
		StartedAt:       time.Now().UTC(),
		TranscriptPath:  in.TranscriptPath,
	}

	if isRepo, err := git.IsRepo(in.Cwd); err == nil && isRepo {
		meta.GitTracked = true
		if sha, err := git.HeadSHA(in.Cwd); err == nil {
			meta.BaseSHA = sha
		}
	}

	return storage.WriteSessionMeta(sDir, meta)
}
