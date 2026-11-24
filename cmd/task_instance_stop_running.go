package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/spf13/cobra"
)

const runningTaskState = "RUNNING_EXECUTION"

type stopRunningOptions struct {
	projectCode  int64
	workflowID   int
	filters      url.Values
	dryRun       bool
	wait         bool
	waitTimeout  time.Duration
	pollInterval time.Duration
	quietPeriod  time.Duration
}

type taskStopTarget struct {
	Kind    string `json:"kind"`
	ID      int    `json:"id"`
	TaskIDs []int  `json:"task_instance_ids"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

type stopRunningResult struct {
	DryRun        bool                        `json:"dry_run"`
	Verified      bool                        `json:"verified"`
	Scans         int                         `json:"scans"`
	RunningTotal  int                         `json:"running_total"`
	Remaining     []dsapi.TaskInstanceSummary `json:"remaining"`
	QueuedByState map[string]int              `json:"queued_by_state,omitempty"`
	Targets       []*taskStopTarget           `json:"targets"`
}

func newTaskInstanceStopRunningCmd(flags *apiFlags) *cobra.Command {
	var opts stopRunningOptions
	c := &cobra.Command{
		Use:   "stop-running",
		Short: "Stop running task instances in a project.",
		Long: "Stop STREAM tasks directly and stop the containing workflow instance for BATCH tasks. " +
			"A workflow STOP also cancels its other active or downstream tasks. " +
			"Use --dry-run to preview the targets. With --wait, stop newly running matches and verify a quiet period.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.projectCode <= 0 {
				return fmt.Errorf("--project-code must be positive")
			}
			if cmd.Flags().Changed("workflow-instance-id") && opts.workflowID <= 0 {
				return fmt.Errorf("--workflow-instance-id must be positive")
			}
			if opts.dryRun && opts.wait {
				return fmt.Errorf("--dry-run and --wait are mutually exclusive")
			}
			if opts.wait && (opts.pollInterval <= 0 || opts.quietPeriod <= 0 || opts.waitTimeout <= opts.quietPeriod) {
				return fmt.Errorf("--poll-interval and --quiet-period must be positive and --wait-timeout must exceed --quiet-period")
			}
			opts.filters = formValues("stateType", runningTaskState)
			if opts.workflowID > 0 {
				opts.filters.Set("workflowInstanceId", strconv.Itoa(opts.workflowID))
			}
			return runStopRunning(cmd, *flags, opts)
		},
	}
	c.Flags().Int64Var(&opts.projectCode, "project-code", 0, "Project code")
	c.Flags().IntVar(&opts.workflowID, "workflow-instance-id", 0, "Limit matches to a workflow instance")
	c.Flags().BoolVar(&opts.dryRun, "dry-run", false, "List targets without submitting STOP requests")
	c.Flags().BoolVar(&opts.wait, "wait", false, "Stop new matches and verify that running and queued tasks have cleared")
	c.Flags().DurationVar(&opts.waitTimeout, "wait-timeout", 30*time.Minute, "Maximum time to submit, poll, and verify with --wait")
	c.Flags().DurationVar(&opts.pollInterval, "poll-interval", 5*time.Second, "Time between scans with --wait")
	c.Flags().DurationVar(&opts.quietPeriod, "quiet-period", 90*time.Second, "Require this long with no running tasks before checking the queue")
	return c
}

func runStopRunning(cmd *cobra.Command, flags apiFlags, opts stopRunningOptions) error {
	const command = "task-instance.stop-running"
	client, profile, err := apiClient(flags)
	if err != nil {
		writeAPIError(cmd, command, "CONFIG_ERROR", err)
		return err
	}
	timeout := profile.Timeout
	if opts.wait {
		timeout = opts.waitTimeout
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := stopRunningTasks(ctx, client, opts)
	if err == nil {
		failed := 0
		for _, target := range result.Targets {
			if target.Status == "error" {
				failed++
			}
		}
		if failed > 0 {
			err = fmt.Errorf("%d stop target(s) returned errors", failed)
		}
	}
	e := output.NewEnvelope(command)
	e.Summary = map[string]any{"cluster": profile.Name, "api_url": profile.APIURL, "project_code": opts.projectCode}
	e.Data = result
	if err != nil {
		e.WithError(output.EnvelopeError{Code: "DS_API_ERROR", Message: err.Error()})
	}
	return errors.Join(err, e.Write(cmd.OutOrStdout()))
}

func stopRunningTasks(ctx context.Context, client *dsapi.Client, opts stopRunningOptions) (stopRunningResult, error) {
	result := stopRunningResult{DryRun: opts.dryRun, Remaining: []dsapi.TaskInstanceSummary{}, Targets: []*taskStopTarget{}}
	type targetKey struct {
		kind string
		id   int
	}
	targets := make(map[targetKey]*taskStopTarget)
	seenTasks := make(map[int]bool)
	var quietSince time.Time
	for {
		tasks, runningTotal, err := runningTasks(ctx, client, opts)
		if err != nil {
			return result, err
		}
		result.Scans++
		result.RunningTotal = runningTotal
		result.Remaining = make([]dsapi.TaskInstanceSummary, 0, len(tasks))
		var pending []*taskStopTarget
		for _, task := range tasks {
			if task.State != runningTaskState || (opts.workflowID > 0 && task.WorkflowInstanceID != opts.workflowID) {
				continue
			}
			result.Remaining = append(result.Remaining, task)
			if seenTasks[task.ID] {
				continue
			}
			seenTasks[task.ID] = true
			key := targetKey{"task", task.ID}
			if task.TaskExecuteType == "BATCH" && task.WorkflowInstanceID > 0 {
				key = targetKey{"workflow", task.WorkflowInstanceID}
			}
			target := targets[key]
			if target == nil {
				target = &taskStopTarget{Kind: key.kind, ID: key.id, Status: "planned"}
				targets[key] = target
				result.Targets = append(result.Targets, target)
				if key.kind == "task" && task.TaskExecuteType != "STREAM" {
					target.Status = "error"
					target.Error = fmt.Sprintf("task %d has unsupported execution type %q or no BATCH workflow instance ID", task.ID, task.TaskExecuteType)
				} else {
					pending = append(pending, target)
				}
			}
			target.TaskIDs = append(target.TaskIDs, task.ID)
		}
		result.RunningTotal = max(result.RunningTotal, len(result.Remaining))
		if opts.dryRun {
			return result, nil
		}
		submitTaskStops(ctx, client, opts.projectCode, pending)
		if !opts.wait {
			return result, ctx.Err()
		}
		if result.RunningTotal > 0 {
			quietSince = time.Time{}
			result.QueuedByState = nil
		} else if quietSince.IsZero() {
			quietSince = time.Now()
		} else if time.Since(quietSince) >= opts.quietPeriod {
			result.QueuedByState, err = queuedTaskCounts(ctx, client, opts.projectCode, opts.filters)
			if err != nil {
				return result, err
			}
			if len(result.QueuedByState) == 0 {
				total, err := taskCount(ctx, client, opts.projectCode, opts.filters, runningTaskState)
				if err != nil {
					return result, err
				}
				if total == 0 {
					result.Verified = true
					return result, nil
				}
				result.RunningTotal = total
				quietSince = time.Time{}
				continue
			}
		}
		timer := time.NewTimer(opts.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, fmt.Errorf("verify task stops (%d running at the last scan): %w", result.RunningTotal, ctx.Err())
		case <-timer.C:
		}
	}
}

func taskExecuteTypes(filters url.Values) []string {
	if filters.Has("workflowInstanceId") {
		// DS 3.4.1 lists standalone STREAM tasks without a workflow instance filter.
		return []string{"BATCH"}
	}
	return []string{"BATCH", "STREAM"}
}

func runningTasks(ctx context.Context, client *dsapi.Client, opts stopRunningOptions) ([]dsapi.TaskInstanceSummary, int, error) {
	var tasks []dsapi.TaskInstanceSummary
	var total int
	for _, kind := range taskExecuteTypes(opts.filters) {
		rows, count, err := runningTasksOfType(ctx, client, opts.projectCode, opts.filters, kind)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, rows...)
		total += count
	}
	return tasks, total, nil
}

func runningTasksOfType(ctx context.Context, client *dsapi.Client, projectCode int64, filters url.Values, kind string) ([]dsapi.TaskInstanceSummary, int, error) {
	query := copyTaskFilters(filters)
	query.Set("taskExecuteType", kind)
	page, err := dsapi.ListAllTaskInstances(ctx, client, projectCode, query, 100)
	if err != nil {
		return nil, 0, fmt.Errorf("list %s tasks: %w", kind, err)
	}
	rows, err := dsapi.SummarizeTaskInstances(page.TotalList)
	return rows, page.Total, err
}

func queuedTaskCounts(ctx context.Context, client *dsapi.Client, projectCode int64, filters url.Values) (map[string]int, error) {
	queue := make(map[string]int)
	for _, state := range []string{"SUBMITTED_SUCCESS", "DISPATCH", "DELAY_EXECUTION"} {
		count, err := taskCount(ctx, client, projectCode, filters, state)
		if err != nil {
			return queue, err
		}
		if count > 0 {
			queue[state] = count
		}
	}
	return queue, nil
}

func taskCount(ctx context.Context, client *dsapi.Client, projectCode int64, filters url.Values, state string) (int, error) {
	query := copyTaskFilters(filters)
	query.Set("stateType", state)
	total := 0
	for _, kind := range taskExecuteTypes(filters) {
		query.Set("taskExecuteType", kind)
		resp, err := dsapi.QueryTaskInstances(ctx, client, projectCode, query, 1, 1)
		if err != nil {
			return 0, fmt.Errorf("check %s %s tasks: %w", kind, state, err)
		}
		page, err := dsapi.DecodeTaskInstancePage(resp.Body)
		if err != nil {
			return 0, err
		}
		total += max(page.Total, len(page.TotalList))
	}
	return total, nil
}

func copyTaskFilters(filters url.Values) url.Values {
	query := url.Values{}
	for key, values := range filters {
		query[key] = values
	}
	return query
}

func submitTaskStops(ctx context.Context, client *dsapi.Client, projectCode int64, targets []*taskStopTarget) {
	// Listing the tasks has already authenticated the client, including with username/password.
	var workers sync.WaitGroup
	jobs := make(chan *taskStopTarget, len(targets))
	for _, target := range targets {
		jobs <- target
	}
	close(jobs)
	for range min(8, len(targets)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for target := range jobs {
				status, err := submitTaskStop(ctx, client, projectCode, target)
				target.Status = status
				if err != nil {
					target.Status, target.Error = "error", err.Error()
				}
			}
		}()
	}
	workers.Wait()
}

func submitTaskStop(ctx context.Context, client *dsapi.Client, projectCode int64, target *taskStopTarget) (string, error) {
	if target.Kind == "task" {
		_, err := dsapi.StopStreamTaskInstance(ctx, client, projectCode, target.ID)
		return "accepted", err
	}
	_, state, err := getWorkflowInstanceState(ctx, client, projectCode, target.ID)
	if err != nil {
		return "", err
	}
	if status := settledStopRequest(state); status != "" {
		return status, nil
	}
	_, err = controlWorkflowInstance(ctx, client, projectCode, target.ID, "STOP")
	if err == nil {
		return "accepted", nil
	}
	// A workflow can finish or enter READY_STOP between the GET and POST.
	_, state, stateErr := getWorkflowInstanceState(ctx, client, projectCode, target.ID)
	if stateErr == nil {
		if status := settledStopRequest(state); status != "" {
			return status, nil
		}
	}
	return "", err
}

func settledStopRequest(state string) string {
	switch state {
	case "READY_STOP":
		return "already_stopping"
	case "STOP", "SUCCESS", "FAILURE", "PAUSE":
		return "terminal"
	default:
		return ""
	}
}
