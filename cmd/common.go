package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ds-cli/ds-cli/internal/config"
	"github.com/ds-cli/ds-cli/internal/local"
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/ds-cli/ds-cli/internal/remote"
	"github.com/ds-cli/ds-cli/internal/runlog"
	"github.com/spf13/cobra"
)

type runCtx struct {
	Cfg        *config.Config
	ConfigPath string
	Run        *runlog.Run
	Runner     local.Runner
	RemotePool *remote.Pool
	Remote     remote.Runner
	Command    string
}

func prepare(cmd *cobra.Command, command string) (*runCtx, error) {
	cfgFlag, _ := cmd.Flags().GetString("config")
	cfgPath, source, err := config.Resolve(cfgFlag)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "using config: %s (%s)\n", cfgPath, source)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	run, err := runlog.New(runlog.DefaultRoot(), command)
	if err != nil {
		return nil, err
	}
	rc := &runCtx{
		Cfg:        cfg,
		ConfigPath: cfgPath,
		Run:        run,
		Runner:     local.Runner{Log: run},
		Command:    command,
	}
	if cfg.Distributed() {
		pool := remote.NewPool(cfg)
		rc.RemotePool = pool
		rc.Remote = remote.Runner{Pool: pool, Parallelism: cfg.SSH.Parallelism, Log: run}
	}
	return rc, nil
}

func (c *runCtx) envelope(command string) *output.Envelope {
	e := output.NewEnvelope(command)
	e.ConfigPath = c.ConfigPath
	e.RunID = c.Run.ID
	return e
}

func (c *runCtx) runStep(ctx context.Context, env *output.Envelope, name, script string) bool {
	fmt.Fprintf(os.Stderr, "[%s] running\n", name)
	res := c.Runner.Run(ctx, name, script)
	step := output.StepResult{Name: name, OK: res.OK, ElapsedMs: res.Elapsed.Milliseconds()}
	if !res.OK {
		step.Message = stepMessage(res)
	}
	env.AddStep(step)
	return res.OK
}

func (c *runCtx) runRemoteStep(ctx context.Context, env *output.Envelope, name string, hosts []string, scriptFor func(string) string) bool {
	if len(hosts) == 0 {
		return true
	}
	fmt.Fprintf(os.Stderr, "[%s] running on %v\n", name, hosts)
	ok := true
	for _, host := range hosts {
		task := remote.Task{Name: name, Cmd: scriptFor(host)}
		for _, res := range c.Remote.Run(ctx, []string{host}, task) {
			step := output.StepResult{Name: name, Host: res.Host, OK: res.OK, ElapsedMs: res.Elapsed.Milliseconds()}
			if !res.OK {
				step.Message = remoteStepMessage(res)
				ok = false
			}
			env.AddStep(step)
		}
	}
	return ok
}

func (c *runCtx) runRemoteSameStep(ctx context.Context, env *output.Envelope, name string, hosts []string, script string) bool {
	return c.runRemoteStep(ctx, env, name, hosts, func(string) string { return script })
}

func stepMessage(res local.Result) string {
	if res.Err != nil {
		if res.Stderr != "" {
			return fmt.Sprintf("%v: %s", res.Err, trimForMessage(res.Stderr))
		}
		return res.Err.Error()
	}
	return trimForMessage(res.Stderr)
}

func remoteStepMessage(res remote.Result) string {
	if res.Err != nil {
		if res.Stderr != "" {
			return fmt.Sprintf("%v: %s", res.Err, trimForMessage(res.Stderr))
		}
		return res.Err.Error()
	}
	if res.Stderr != "" {
		return trimForMessage(res.Stderr)
	}
	return fmt.Sprintf("exit=%d", res.ExitCode)
}

func trimForMessage(s string) string {
	if len(s) > 500 {
		return s[:500] + "..."
	}
	return s
}

func writeEnvelope(e *output.Envelope) {
	_ = e.Write(os.Stdout)
}

func finish(c *runCtx, e *output.Envelope) error {
	if c.RemotePool != nil {
		c.RemotePool.Close()
	}
	_ = c.Run.SaveResult(e)
	writeEnvelope(e)
	if !e.OK {
		if e.Error != nil {
			return fmt.Errorf("[%s] %s", e.Error.Code, e.Error.Message)
		}
		for _, step := range e.Steps {
			if !step.OK {
				return fmt.Errorf("[%s] %s", step.Name, step.Message)
			}
		}
		return fmt.Errorf("command failed")
	}
	return nil
}

func commandCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Minute)
}
