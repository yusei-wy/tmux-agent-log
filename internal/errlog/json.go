package errlog

import "encoding/json"

func unmarshal(raw []byte, v any) error {
	return json.Unmarshal(raw, v)
}
