package hook

import (
	"encoding/json"
	"fmt"
	"io"
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
	if err := json.NewDecoder(stdin).Decode(&in); err != nil {
		return fmt.Errorf("decode turn_end input: %w", err)
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

	turnsPath := filepath.Join(sDir, "turns.jsonl")
	turns, err := storage.ReadTurns(turnsPath)
	if err != nil {
		return fmt.Errorf("read turns %s: %w", turnsPath, err)
	}

	var openTurn *storage.Turn
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].Status == storage.TurnStatusOpen {
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
		Status:  storage.TurnStatusDone,
	}

	if meta.GitTracked {
		diff, err := git.DiffSince(in.Cwd, openTurn.HeadSHAPre)
		if err == nil && strings.TrimSpace(diff) != "" {
			if err := storage.WriteTurnDiff(sDir, openTurn.ID, []byte(diff)); err != nil {
				return fmt.Errorf("write turn diff %s: %w", openTurn.ID, err)
			}
			close.DiffPath = storage.TurnDiffRelPath(openTurn.ID)
		}
		if sha, err := git.HeadSHA(in.Cwd); err == nil {
			close.HeadSHA = sha
		}
	}

	return storage.AppendTurnClose(turnsPath, close)
}
