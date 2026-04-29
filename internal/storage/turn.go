package storage

import (
	"encoding/json"
	"sort"
)

func AppendTurnOpen(path string, t TurnOpen) error {
	t.Phase = TurnPhaseOpen
	return AppendJSONL(path, t)
}

func AppendTurnClose(path string, t TurnClose) error {
	t.Phase = TurnPhaseClose
	return AppendJSONL(path, t)
}

func ReadTurns(path string) ([]Turn, error) {
	turns := map[string]*Turn{}

	err := ReadJSONL(path, func(raw []byte) error {
		var head struct {
			ID    string    `json:"id"`
			Phase TurnPhase `json:"phase"`
		}
		if err := json.Unmarshal(raw, &head); err != nil {
			return nil
		}
		if head.ID == "" {
			return nil
		}

		switch head.Phase {
		case TurnPhaseOpen:
			var o TurnOpen
			if err := json.Unmarshal(raw, &o); err != nil {
				return nil
			}
			t, ok := turns[o.ID]
			if !ok {
				t = &Turn{ID: o.ID, Status: TurnStatusOpen}
				turns[o.ID] = t
			}
			t.StartedAt = o.StartedAt
			t.UserPromptPreview = o.UserPromptPreview
			t.HeadSHAPre = o.HeadSHAPre
			t.TranscriptMessageID = o.TranscriptMessageID
		case TurnPhaseClose:
			var c TurnClose
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil
			}
			t, ok := turns[c.ID]
			if !ok {
				t = &Turn{ID: c.ID, Status: TurnStatusOpen}
				turns[c.ID] = t
			}
			t.EndedAt = c.EndedAt
			t.AssistantSummaryPreview = c.AssistantSummaryPreview
			t.HeadSHA = c.HeadSHA
			t.DiffPath = c.DiffPath
			t.Status = c.Status
			t.ErrorMessage = c.ErrorMessage
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]Turn, 0, len(turns))
	for _, t := range turns {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.Before(out[j].StartedAt)
	})
	return out, nil
}
