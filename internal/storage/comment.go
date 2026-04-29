package storage

import (
	"encoding/json"
	"sort"
	"time"
)

type commentRecord struct {
	Comment
	Deleted bool      `json:"deleted,omitempty"`
	SetSent time.Time `json:"set_sent,omitempty"`
}

func AppendComment(path string, c Comment) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	return AppendJSONL(path, commentRecord{Comment: c})
}

func MarkCommentsSent(path string, ids []string, ts time.Time) error {
	for _, id := range ids {
		rec := commentRecord{Comment: Comment{ID: id}, SetSent: ts}
		if err := AppendJSONL(path, rec); err != nil {
			return err
		}
	}
	return nil
}

func DeleteComment(path, id string) error {
	return AppendJSONL(path, commentRecord{Comment: Comment{ID: id}, Deleted: true})
}

func ReadComments(path string) ([]Comment, error) {
	merged := map[string]*Comment{}
	deleted := map[string]bool{}
	order := map[string]int{}

	idx := 0
	err := ReadJSONL(path, func(raw []byte) error {
		var rec commentRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return nil
		}
		if rec.ID == "" {
			return nil
		}
		if rec.Deleted {
			deleted[rec.ID] = true
			delete(merged, rec.ID)
			return nil
		}
		c, ok := merged[rec.ID]
		if !ok {
			if deleted[rec.ID] {
				return nil
			}
			cp := rec.Comment
			merged[rec.ID] = &cp
			order[rec.ID] = idx
			idx++
			c = merged[rec.ID]
		}
		if !rec.SetSent.IsZero() {
			c.SentAt = rec.SetSent
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]Comment, 0, len(merged))
	for _, c := range merged {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return order[out[i].ID] < order[out[j].ID]
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func UnsentComments(path string) ([]Comment, error) {
	all, err := ReadComments(path)
	if err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(all))
	for _, c := range all {
		if c.SentAt.IsZero() {
			out = append(out, c)
		}
	}
	return out, nil
}
