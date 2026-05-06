package storage

import "encoding/json"

func AppendEvent(path string, e Event) error {
	return AppendJSONL(path, e)
}

func ReadEvents(path, turnID string) ([]Event, error) {
	var out []Event

	err := ReadJSONL(path, func(raw []byte) error {
		var e Event
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil
		}

		if turnID != "" && e.TurnID != turnID {
			return nil
		}

		out = append(out, e)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}
