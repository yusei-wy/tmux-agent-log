package storage

import "encoding/json"

// LatestOpenTurnID は turns.jsonl を 1 パスでスキャンし、close されていない
// 最新の open turn の id を返す。tool-pre/tool-post hook が tool 呼出しごとに
// 走るため、ReadTurns の map+sort コストを避けたい用途で使う。
//
// 該当なしのときは空文字列+nil。ファイル不在も空文字列+nil。
func LatestOpenTurnID(path string) (string, error) {
	open := []string{}
	closed := map[string]bool{}

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
			open = append(open, head.ID)
		case TurnPhaseClose:
			closed[head.ID] = true
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	for i := len(open) - 1; i >= 0; i-- {
		if !closed[open[i]] {
			return open[i], nil
		}
	}
	return "", nil
}
