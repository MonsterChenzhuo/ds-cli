package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWorkflowInstanceCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:     "workflow-instance",
		Aliases: []string{"wfi"},
		Short:   "Inspect and control DolphinScheduler workflow instances.",
	}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newWorkflowInstanceListCmd(&flags))
	cmd.AddCommand(newWorkflowInstanceBackfillStatusCmd(&flags))
	cmd.AddCommand(newWorkflowInstanceGetCmd(&flags))
	cmd.AddCommand(newWorkflowInstanceTasksCmd(&flags))
	cmd.AddCommand(newWorkflowInstanceControlCmd(&flags))
	cmd.AddCommand(newWorkflowInstanceDeleteCmd(&flags))
	return cmd
}

func newWorkflowInstanceListCmd(flags *apiFlags) *cobra.Command {
	var projectCode, workflowCode int64
	var stateType, startDate, endDate, executorName, search, host, cmdType string
	var pageNo, pageSize int
	var compact, all bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List workflow instances in a project.",
		Long: "list returns one page of workflow instances. --all pages through every instance and\n" +
			"--compact drops the huge per-instance fields (commandParam/globalParams/stateHistory/\n" +
			"locations/...), leaving id/name/state/scheduleTime/times so an agent can scan the list\n" +
			"without downloading hundreds of KB per page. --cmd-type filters the result client-side\n" +
			"(e.g. COMPLEMENT_DATA for backfill instances).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			if !compact && !all && cmdType == "" {
				return apiRun(cmd, *flags, "workflow-instance.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
					values := formValues(
						"searchVal", search,
						"executorName", executorName,
						"stateType", stateType,
						"host", host,
						"startDate", startDate,
						"endDate", endDate,
						"pageNo", strconv.Itoa(pageNo),
						"pageSize", strconv.Itoa(pageSize),
					)
					if workflowCode != 0 {
						values.Set("workflowDefinitionCode", strconv.FormatInt(workflowCode, 10))
					}
					return client.Form(ctx, http.MethodGet,
						fmt.Sprintf("/projects/%d/workflow-instances", projectCode), values)
				})
			}

			client, profile, err := apiClient(*flags)
			if err != nil {
				writeAPIError(cmd, "workflow-instance.list", "CONFIG_ERROR", err)
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
			defer cancel()

			var rows []dsapi.WorkflowInstanceSummary
			total := 0
			options := dsapi.WorkflowInstanceListOptions{
				WorkflowCode: workflowCode,
				StateType:    stateType,
				StartDate:    startDate,
				EndDate:      endDate,
				ExecutorName: executorName,
				Search:       search,
				Host:         host,
			}
			if all {
				rows, err = dsapi.ListAllWorkflowInstancesWithOptions(ctx, client, projectCode, options, pageSize)
				if err != nil {
					writeAPIError(cmd, "workflow-instance.list", "DS_API_ERROR", err)
					return err
				}
				total = len(rows)
			} else {
				rows, total, err = dsapi.ListWorkflowInstancesWithOptions(ctx, client, projectCode, options, pageNo, pageSize)
				if err != nil {
					writeAPIError(cmd, "workflow-instance.list", "DS_API_ERROR", err)
					return err
				}
			}
			if cmdType != "" {
				rows = filterWorkflowInstancesByCmdType(rows, cmdType)
			}
			if compact {
				for i := range rows {
					rows[i].CommandParam = ""
				}
			}
			e := output.NewEnvelope("workflow-instance.list")
			e.Summary = map[string]any{
				"cluster":     profile.Name,
				"api_url":     profile.APIURL,
				"http_status": 200,
			}
			e.Data = map[string]any{
				"code": 0,
				"msg":  "success",
				"data": map[string]any{
					"total":     total,
					"returned":  len(rows),
					"pageNo":    pageNo,
					"pageSize":  pageSize,
					"totalList": rows,
				},
			}
			return e.Write(cmd.OutOrStdout())
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().Int64Var(&workflowCode, "workflow-code", 0, "Filter by workflow definition code")
	c.Flags().StringVar(&stateType, "state-type", "", "Filter by execution status, e.g. SUCCESS/FAILURE/RUNNING_EXECUTION")
	c.Flags().StringVar(&startDate, "start-date", "", "Filter range start, e.g. 2026-01-01 00:00:00")
	c.Flags().StringVar(&endDate, "end-date", "", "Filter range end")
	c.Flags().StringVar(&executorName, "executor-name", "", "Filter by executor user name")
	c.Flags().StringVar(&search, "search", "", "Search text")
	c.Flags().StringVar(&host, "host", "", "Filter by worker host")
	c.Flags().StringVar(&cmdType, "cmd-type", "", "Filter client-side by command type, e.g. COMPLEMENT_DATA/SCHEDULER/START_PROCESS")
	c.Flags().BoolVar(&all, "all", false, "Page through every matching instance instead of one page")
	c.Flags().BoolVar(&compact, "compact", false, "Drop huge per-instance fields (commandParam/globalParams/stateHistory/...)")
	c.Flags().IntVar(&pageNo, "page-no", 1, "Page number")
	c.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return c
}

// filterWorkflowInstancesByCmdType keeps instances whose command type matches
// want (COMPLEMENT_DATA / SCHEDULER / START_PROCESS / ...).
func filterWorkflowInstancesByCmdType(rows []dsapi.WorkflowInstanceSummary, want string) []dsapi.WorkflowInstanceSummary {
	want = strings.ToUpper(strings.TrimSpace(want))
	out := make([]dsapi.WorkflowInstanceSummary, 0, len(rows))
	for _, r := range rows {
		if strings.ToUpper(r.CmdTypeIfComplement) == want || strings.ToUpper(r.CommandType) == want {
			out = append(out, r)
		}
	}
	return out
}

func newWorkflowInstanceGetCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "get <instance-id>",
		Short: "Get a workflow instance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "instance-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "workflow-instance.get", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet,
					fmt.Sprintf("/projects/%d/workflow-instances/%d", projectCode, id), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newWorkflowInstanceTasksCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "tasks <instance-id>",
		Short: "List tasks belonging to a workflow instance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "instance-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "workflow-instance.tasks", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet,
					fmt.Sprintf("/projects/%d/workflow-instances/%d/tasks", projectCode, id), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

var workflowInstanceControlMap = map[string]string{
	"STOP":           "STOP",
	"PAUSE":          "PAUSE",
	"RESUME":         "RECOVER_SUSPENDED_PROCESS",
	"RERUN":          "REPEAT_RUNNING",
	"RECOVER-FAILED": "START_FAILURE_TASK_PROCESS",
}

func newWorkflowInstanceControlCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	var action string
	c := &cobra.Command{
		Use:   "control <instance-id>",
		Short: "Control a workflow instance: STOP, PAUSE, RESUME, RERUN, RECOVER-FAILED.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "instance-id")
			if err != nil {
				return err
			}
			executeType, ok := workflowInstanceControlMap[strings.ToUpper(strings.TrimSpace(action))]
			if !ok {
				return fmt.Errorf("--type must be one of STOP, PAUSE, RESUME, RERUN, RECOVER-FAILED")
			}
			return apiRun(cmd, *flags, "workflow-instance.control", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost,
					fmt.Sprintf("/projects/%d/executors/execute", projectCode),
					formValues("workflowInstanceId", strconv.Itoa(id), "executeType", executeType))
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().StringVar(&action, "type", "", "Action: STOP, PAUSE, RESUME, RERUN, RECOVER-FAILED")
	_ = c.MarkFlagRequired("type")
	return c
}

func newWorkflowInstanceDeleteCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "delete <instance-id>",
		Short: "Delete a workflow instance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "instance-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "workflow-instance.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodDelete,
					fmt.Sprintf("/projects/%d/workflow-instances/%d", projectCode, id), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newWorkflowInstanceBackfillStatusCmd(flags *apiFlags) *cobra.Command {
	var projectCode, workflowCode int64
	var startDate, endDate, dateList, dateListFile, startedOn string
	var details, includeNonComplement bool
	c := &cobra.Command{
		Use:   "backfill-status",
		Short: "Check which days of a date range a workflow's backfill (COMPLEMENT_DATA) actually covered.",
		Long: "backfill-status pages through every workflow instance, reads the schedule dates each\n" +
			"backfill instance covers (commandParam.backfillTimeList and actual scheduleTime) and reports\n" +
			"which days of the expected range are missing, not SUCCESS, or duplicated.\n\n" +
			"This replaces the manual dance of listing all pages, parsing commandParam and diffing\n" +
			"dates by hand.\n\n" +
			"Expected range: --start-date/--end-date, or --date-list/--date-list-file for an\n" +
			"explicit set of days. --started-on restricts the scan to instances started on those\n" +
			"dates (e.g. one backfill campaign), otherwise every backfill instance is considered.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			if workflowCode == 0 {
				return fmt.Errorf("--workflow-code is required")
			}
			expected, err := resolveExpectedDates(startDate, endDate, dateList, dateListFile)
			if err != nil {
				return err
			}
			startedOnDates := splitDateList(startedOn)

			client, profile, err := apiClient(*flags)
			if err != nil {
				writeAPIError(cmd, "workflow-instance.backfill-status", "CONFIG_ERROR", err)
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
			defer cancel()

			all, err := dsapi.ListAllWorkflowInstances(ctx, client, projectCode, workflowCode, 200)
			if err != nil {
				writeAPIError(cmd, "workflow-instance.backfill-status", "DS_API_ERROR", err)
				return err
			}
			scanned := len(all)
			matched := make([]dsapi.WorkflowInstanceSummary, 0, len(all))
			for _, inst := range all {
				if !includeNonComplement && !inst.IsComplement() {
					continue
				}
				if len(startedOnDates) > 0 && !containsString(startedOnDates, dsapi.NormalizeDate(inst.StartTime)) {
					continue
				}
				matched = append(matched, inst)
			}

			cov := dsapi.ComputeBackfillCoverage(expected, matched)
			complete := len(cov.MissingDates) == 0 && len(cov.NotExecutedDates) == 0 && len(cov.NotSuccessDates) == 0
			if !details {
				cov.Dates = nil
			}

			e := output.NewEnvelope("workflow-instance.backfill-status")
			e.Summary = map[string]any{
				"cluster":     profile.Name,
				"api_url":     profile.APIURL,
				"http_status": 200,
			}
			e.Data = map[string]any{
				"project_code":       projectCode,
				"workflow_code":      workflowCode,
				"range_start":        firstOr(expected, ""),
				"range_end":          lastOr(expected, ""),
				"started_on":         startedOnDates,
				"instances_scanned":  scanned,
				"instances_matched":  len(matched),
				"expected_days":      cov.ExpectedDays,
				"executed_days":      cov.ExecutedDays,
				"success_days":       cov.SuccessDays,
				"missing_days":       len(cov.MissingDates),
				"missing_dates":      cov.MissingDates,
				"not_executed_days":  len(cov.NotExecutedDates),
				"not_executed_dates": cov.NotExecutedDates,
				"not_success_dates":  cov.NotSuccessDates,
				"duplicate_dates":    cov.DuplicateDates,
				"extra_dates":        cov.ExtraDates,
				"state_counts":       cov.StateCounts,
				"complete":           complete,
				"dates":              cov.Dates,
			}
			return e.Write(cmd.OutOrStdout())
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().Int64Var(&workflowCode, "workflow-code", 0, "Workflow definition code")
	c.Flags().StringVar(&startDate, "start-date", "", "Expected range start, e.g. 2025-01-01")
	c.Flags().StringVar(&endDate, "end-date", "", "Expected range end (inclusive), e.g. 2026-09-20")
	c.Flags().StringVar(&dateList, "date-list", "", "Expected dates (comma separated) instead of a range")
	c.Flags().StringVar(&dateListFile, "date-list-file", "", "Read expected dates from a file (comma/newline separated)")
	c.Flags().StringVar(&startedOn, "started-on", "", "Only scan instances whose start date is in this comma-separated list, e.g. 2026-09-21,2026-09-22")
	c.Flags().BoolVar(&details, "details", false, "Include per-date state/instance ids")
	c.Flags().BoolVar(&includeNonComplement, "include-non-complement", false, "Also count non-backfill (scheduled/manual) instances")
	return c
}

func resolveExpectedDates(startDate, endDate, dateList, dateListFile string) ([]string, error) {
	entries := splitDateList(dateList)
	if dateListFile != "" {
		if dateList != "" {
			return nil, fmt.Errorf("--date-list and --date-list-file are mutually exclusive")
		}
		b, err := os.ReadFile(dateListFile)
		if err != nil {
			return nil, fmt.Errorf("read --date-list-file: %w", err)
		}
		entries = splitDateList(string(b))
	}
	if len(entries) > 0 {
		if startDate != "" || endDate != "" {
			return nil, fmt.Errorf("--date-list/--date-list-file and --start-date/--end-date are mutually exclusive")
		}
		out := make([]string, 0, len(entries))
		for _, e := range entries {
			d := dsapi.NormalizeDate(e)
			if d == "" {
				return nil, fmt.Errorf("invalid date %q: expect yyyy-MM-dd or yyyy-MM-dd HH:mm:ss", e)
			}
			out = append(out, d)
		}
		return out, nil
	}
	if startDate == "" || endDate == "" {
		return nil, fmt.Errorf("provide --start-date/--end-date, or --date-list/--date-list-file")
	}
	return dsapi.DateRange(startDate, endDate)
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func firstOr(list []string, fallback string) string {
	if len(list) == 0 {
		return fallback
	}
	return list[0]
}

func lastOr(list []string, fallback string) string {
	if len(list) == 0 {
		return fallback
	}
	return list[len(list)-1]
}
