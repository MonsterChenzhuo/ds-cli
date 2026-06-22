package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ds-cli/ds-cli/internal/dsapi"
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
	cmd.AddCommand(newWorkflowInstanceGetCmd(&flags))
	cmd.AddCommand(newWorkflowInstanceTasksCmd(&flags))
	cmd.AddCommand(newWorkflowInstanceControlCmd(&flags))
	cmd.AddCommand(newWorkflowInstanceDeleteCmd(&flags))
	return cmd
}

func newWorkflowInstanceListCmd(flags *apiFlags) *cobra.Command {
	var projectCode, workflowCode int64
	var stateType, startDate, endDate, executorName, search, host string
	var pageNo, pageSize int
	c := &cobra.Command{
		Use:   "list",
		Short: "List workflow instances in a project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
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
	c.Flags().IntVar(&pageNo, "page-no", 1, "Page number")
	c.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return c
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
