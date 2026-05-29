package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Create, release, offline, and delete DS task-style single-task workflows.",
		Long:  "The task command creates a DolphinScheduler workflow definition containing one SHELL or PYTHON task node, then reuses workflow release/delete APIs for online, offline, and delete operations.",
	}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newTaskCreateCmd(&flags))
	cmd.AddCommand(newTaskReleaseCmd(&flags, "online", "ONLINE"))
	cmd.AddCommand(newTaskReleaseCmd(&flags, "offline", "OFFLINE"))
	cmd.AddCommand(newTaskDeleteCmd(&flags))
	cmd.AddCommand(newTaskGetCmd(&flags))
	cmd.AddCommand(newTaskListCmd(&flags))
	return cmd
}

func newTaskCreateCmd(flags *apiFlags) *cobra.Command {
	var projectCode, environmentCode int64
	var workflowName, description, taskType, script, scriptFile, workerGroup string
	c := &cobra.Command{
		Use:   "create <task-name>",
		Short: "Create an offline single-task workflow definition.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			if workflowName == "" {
				workflowName = args[0]
			}
			if scriptFile != "" {
				b, err := os.ReadFile(scriptFile)
				if err != nil {
					return err
				}
				script = string(b)
			}
			if script == "" {
				return fmt.Errorf("--script or --script-file is required")
			}
			return apiRun(cmd, *flags, "task.create", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				code, err := generateTaskCode(ctx, client, projectCode)
				if err != nil {
					return nil, err
				}
				form, err := dsapi.SingleTaskWorkflowForm(dsapi.SingleTaskWorkflow{
					ProjectCode:     projectCode,
					WorkflowName:    workflowName,
					Description:     description,
					TaskName:        args[0],
					TaskCode:        code,
					TaskType:        taskType,
					Script:          script,
					WorkerGroup:     workerGroup,
					EnvironmentCode: environmentCode,
				})
				if err != nil {
					return nil, err
				}
				return client.Form(ctx, http.MethodPost, fmt.Sprintf("/projects/%d/workflow-definition", projectCode), form)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().StringVar(&workflowName, "workflow-name", "", "Workflow name; defaults to task name")
	c.Flags().StringVar(&description, "description", "", "Workflow description")
	c.Flags().StringVar(&taskType, "type", "SHELL", "Task type: SHELL or PYTHON")
	c.Flags().StringVar(&script, "script", "", "Task raw script")
	c.Flags().StringVar(&scriptFile, "script-file", "", "Read task raw script from file")
	c.Flags().StringVar(&workerGroup, "worker-group", "default", "Worker group")
	c.Flags().Int64Var(&environmentCode, "environment-code", 0, "Environment code; 0 means unset")
	return c
}

func generateTaskCode(ctx context.Context, client *dsapi.Client, projectCode int64) (int64, error) {
	resp, err := client.Form(ctx, http.MethodGet,
		fmt.Sprintf("/projects/%d/task-definition/gen-task-codes", projectCode),
		formValues("genNum", "1"),
	)
	if err != nil {
		return 0, err
	}
	var decoded any
	if err := json.Unmarshal(resp.Body, &decoded); err != nil {
		return 0, err
	}
	code, ok := findFirstNumber(decoded, "dataList")
	if !ok {
		code, ok = findFirstNumber(decoded, "data")
	}
	if !ok {
		return 0, fmt.Errorf("could not find generated task code in response: %s", string(resp.Body))
	}
	return code, nil
}

func findFirstNumber(v any, key string) (int64, bool) {
	switch t := v.(type) {
	case map[string]any:
		if val, ok := t[key]; ok {
			if n, ok := firstNumber(val); ok {
				return n, true
			}
		}
		for _, val := range t {
			if n, ok := findFirstNumber(val, key); ok {
				return n, true
			}
		}
	}
	return 0, false
}

func firstNumber(v any) (int64, bool) {
	switch t := v.(type) {
	case []any:
		if len(t) == 0 {
			return 0, false
		}
		return firstNumber(t[0])
	case float64:
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	}
	return 0, false
}

func newTaskReleaseCmd(flags *apiFlags, name, state string) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   name + " <workflow-code>",
		Short: name + " a task workflow definition.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "task."+name, func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return releaseWorkflow(ctx, client, projectCode, code, state)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newTaskDeleteCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <workflow-code>",
		Short: "Delete a task workflow definition.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "task.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodDelete, fmt.Sprintf("/v2/workflows/%d", code), nil)
			})
		},
	}
}

func newTaskGetCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <workflow-code>",
		Short: "Get a task workflow definition.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "task.get", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodGet, fmt.Sprintf("/v2/workflows/%d", code), nil)
			})
		},
	}
}

func newTaskListCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "list",
		Short: "List task workflow definitions in a project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			return apiRun(cmd, *flags, "task.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet, fmt.Sprintf("/projects/%d/workflow-definition/list", projectCode), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}
