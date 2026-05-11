package remote

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ds-cli/ds-cli/internal/runlog"
)

type Runner struct {
	Pool        *Pool
	Parallelism int
	Log         *runlog.Run
}

func (r Runner) Run(ctx context.Context, hosts []string, task Task) []Result {
	parallelism := r.Parallelism
	if parallelism <= 0 {
		parallelism = 4
	}
	if task.Timeout == 0 {
		task.Timeout = 10 * time.Minute
	}
	out := make([]Result, len(hosts))
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	for i, host := range hosts {
		i, host := i, host
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			start := time.Now()
			subCtx, cancel := context.WithTimeout(ctx, task.Timeout)
			defer cancel()
			res := r.execute(subCtx, host, task)
			res.Elapsed = time.Since(start)
			out[i] = res
			if r.Log != nil {
				_ = r.Log.WriteFile(fmt.Sprintf("%s.%s.stdout", task.Name, host), []byte(res.Stdout))
				_ = r.Log.WriteFile(fmt.Sprintf("%s.%s.stderr", task.Name, host), []byte(res.Stderr))
			}
		}()
	}
	wg.Wait()
	return out
}

func (r Runner) execute(ctx context.Context, host string, task Task) Result {
	client, err := r.Pool.Get(host)
	if err != nil {
		return Result{Host: host, OK: false, Err: err}
	}
	for _, f := range task.Inline {
		if err := client.WriteFile(f.Remote, f.Content, f.Mode); err != nil {
			return Result{Host: host, OK: false, Err: err}
		}
	}
	if task.Cmd == "" {
		return Result{Host: host, OK: true}
	}
	exec, err := client.Exec(ctx, task.Cmd)
	if err != nil {
		return Result{Host: host, OK: false, Err: err}
	}
	return Result{Host: host, OK: exec.ExitCode == 0, Stdout: exec.Stdout, Stderr: exec.Stderr, ExitCode: exec.ExitCode}
}
