package hook

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/git"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

type turnEndInput struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
}

func RunTurnEnd(stdin io.Reader) error {
	var in turnEndInput
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

	turnsPath := filepath.Join(sDir, "turns.jsonl")
	turns, err := storage.ReadTurns(turnsPath)
	if err != nil {
		return err
	}

	var openTurn *storage.Turn
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].Status == "open" {
			openTurn = &turns[i]
			break
		}
	}
	if openTurn == nil {
		return nil
	}

	close := storage.TurnClose{
		ID:      openTurn.ID,
		EndedAt: time.Now().UTC(),
		Status:  "done",
	}

	if meta.GitTracked {
		diff, err := git.DiffSince(in.Cwd, openTurn.HeadSHAPre)
		if err == nil && strings.TrimSpace(diff) != "" {
			diffsDir := filepath.Join(sDir, "diffs")
			if err := os.MkdirAll(diffsDir, 0o700); err != nil {
				return err
			}
			rel := filepath.Join("diffs", openTurn.ID+".patch")
			if err := os.WriteFile(filepath.Join(sDir, rel), []byte(diff), 0o600); err != nil {
				return err
			}
			close.DiffPath = rel
		}
		if sha, err := git.HeadSHA(in.Cwd); err == nil {
			close.HeadSHA = sha
		}
	}

	return storage.AppendTurnClose(turnsPath, close)
}
