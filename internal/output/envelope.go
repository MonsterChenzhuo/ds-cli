package output

import (
	"encoding/json"
	"io"
)

type StepResult struct {
	Name      string `json:"name"`
	Host      string `json:"host,omitempty"`
	OK        bool   `json:"ok"`
	ElapsedMs int64  `json:"elapsed_ms"`
	Message   string `json:"message,omitempty"`
}

type EnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type Envelope struct {
	Command    string         `json:"command"`
	OK         bool           `json:"ok"`
	Summary    map[string]any `json:"summary,omitempty"`
	Steps      []StepResult   `json:"steps,omitempty"`
	Error      *EnvelopeError `json:"error,omitempty"`
	RunID      string         `json:"run_id,omitempty"`
	ConfigPath string         `json:"config_path,omitempty"`
}

func NewEnvelope(command string) *Envelope {
	return &Envelope{Command: command, OK: true}
}

func (e *Envelope) AddStep(r StepResult) *Envelope {
	if !r.OK {
		e.OK = false
	}
	e.Steps = append(e.Steps, r)
	return e
}

func (e *Envelope) WithError(err EnvelopeError) *Envelope {
	e.OK = false
	e.Error = &err
	return e
}

func (e *Envelope) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(e)
}
