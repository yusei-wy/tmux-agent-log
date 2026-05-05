package storage

import "time"

type TurnPhase string

const (
	TurnPhaseOpen  TurnPhase = "open"
	TurnPhaseClose TurnPhase = "close"
)

type EventPhase string

const (
	EventPhasePre  EventPhase = "pre"
	EventPhasePost EventPhase = "post"
)

type TurnStatus string

const (
	TurnStatusOpen  TurnStatus = "open"
	TurnStatusDone  TurnStatus = "done"
	TurnStatusError TurnStatus = "error"
)

type SessionMeta struct {
	ClaudeSessionID string    `json:"claude_session_id"`
	TmuxPane        string    `json:"tmux_pane,omitempty"`
	Cwd             string    `json:"cwd"`
	Goal            string    `json:"goal,omitempty"`
	BaseSHA         string    `json:"base_sha,omitempty"`
	GitTracked      bool      `json:"git_tracked"`
	StartedAt       time.Time `json:"started_at"`
	TranscriptPath  string    `json:"transcript_path,omitempty"`
}

type TurnOpen struct {
	ID                  string    `json:"id"`
	Phase               TurnPhase `json:"phase"`
	StartedAt           time.Time `json:"started_at"`
	UserPromptPreview   string    `json:"user_prompt_preview,omitempty"`
	HeadSHAPre          string    `json:"head_sha_pre,omitempty"`
	TranscriptMessageID string    `json:"transcript_message_id,omitempty"`
}

type TurnClose struct {
	ID                      string     `json:"id"`
	Phase                   TurnPhase  `json:"phase"`
	EndedAt                 time.Time  `json:"ended_at"`
	AssistantSummaryPreview string     `json:"assistant_summary_preview,omitempty"`
	HeadSHA                 string     `json:"head_sha,omitempty"`
	DiffPath                string     `json:"diff_path,omitempty"`
	Status                  TurnStatus `json:"status"`
	ErrorMessage            string     `json:"error_message,omitempty"`
}

type Turn struct {
	ID                      string     `json:"id"`
	StartedAt               time.Time  `json:"started_at"`
	EndedAt                 *time.Time `json:"ended_at,omitempty"`
	UserPromptPreview       string     `json:"user_prompt_preview,omitempty"`
	AssistantSummaryPreview string     `json:"assistant_summary_preview,omitempty"`
	HeadSHAPre              string     `json:"head_sha_pre,omitempty"`
	HeadSHA                 string     `json:"head_sha,omitempty"`
	DiffPath                string     `json:"diff_path,omitempty"`
	Status                  TurnStatus `json:"status"`
	ErrorMessage            string     `json:"error_message,omitempty"`
	TranscriptMessageID     string     `json:"transcript_message_id,omitempty"`
}

type Event struct {
	ID           string     `json:"id"`
	TurnID       string     `json:"turn_id"`
	TS           time.Time  `json:"ts"`
	Tool         string     `json:"tool"`
	ArgsPreview  string     `json:"args_preview,omitempty"`
	Phase        EventPhase `json:"phase"`
	Success      bool       `json:"success,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

type Comment struct {
	ID        string     `json:"id"`
	File      string     `json:"file"`
	LineStart int        `json:"line_start"`
	LineEnd   int        `json:"line_end"`
	Text      string     `json:"text"`
	CreatedAt time.Time  `json:"created_at"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
}

type ErrEntry struct {
	TS          time.Time `json:"ts"`
	Component   string    `json:"component"`
	Event       string    `json:"event"`
	SessionID   string    `json:"session_id,omitempty"`
	ErrorString string    `json:"error"`
}
