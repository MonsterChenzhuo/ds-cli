package local

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/ds-cli/ds-cli/internal/runlog"
)

type Result struct {
	Name    string
	OK      bool
	Elapsed time.Duration
	Stdout  string
	Stderr  string
	Err     error
}

type Runner struct {
	Log *runlog.Run
}

func (r Runner) Run(ctx context.Context, name, script string) Result {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "bash", "-lc", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := Result{
		Name:    name,
		OK:      err == nil,
		Elapsed: time.Since(start),
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		Err:     err,
	}
	if r.Log != nil {
		_ = r.Log.WriteFile(fmt.Sprintf("%s.stdout", name), []byte(res.Stdout))
		_ = r.Log.WriteFile(fmt.Sprintf("%s.stderr", name), []byte(res.Stderr))
	}
	return res
}
