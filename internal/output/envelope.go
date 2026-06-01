package output

import (
	"encoding/json"
	"io"
)

type EnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type Envelope struct {
	Command string         `json:"command"`
	OK      bool           `json:"ok"`
	Summary map[string]any `json:"summary,omitempty"`
	Data    any            `json:"data,omitempty"`
	Error   *EnvelopeError `json:"error,omitempty"`
}

func NewEnvelope(command string) *Envelope {
	return &Envelope{Command: command, OK: true}
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
