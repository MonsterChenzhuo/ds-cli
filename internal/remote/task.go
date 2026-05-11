package remote

import (
	"os"
	"time"
)

type InlineFile struct {
	Remote  string
	Content []byte
	Mode    os.FileMode
}

type Task struct {
	Name    string
	Cmd     string
	Inline  []InlineFile
	Timeout time.Duration
}

type Result struct {
	Host     string
	OK       bool
	Stdout   string
	Stderr   string
	ExitCode int
	Elapsed  time.Duration
	Err      error
}
