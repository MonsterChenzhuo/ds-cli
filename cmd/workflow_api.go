package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWorkflowCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:     "workflow",
		Aliases: []string{"wf"},
		Short:   "Manage DolphinScheduler workflow definitions through the REST API.",
	}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newWorkflowCreateCmd(&flags))
	cmd.AddCommand(newWorkflowUpdateCmd(&flags))
	cmd.AddCommand(newWorkflowGetCmd(&flags))
	cmd.AddCommand(newWorkflowGetDetailCmd(&flags))
	cmd.AddCommand(newWorkflowListCmd(&flags))
	cmd.AddCommand(newWorkflowReleaseCmd(&flags, "online", "ONLINE"))
	cmd.AddCommand(newWorkflowReleaseCmd(&flags, "offline", "OFFLINE"))
	cmd.AddCommand(newWorkflowDeleteCmd(&flags))
	cmd.AddCommand(newWorkflowPatchTaskCmd(&flags))
	cmd.AddCommand(newWorkflowStartCmd(&flags))
	return cmd
}

func newWorkflowCreateCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	var description, releaseState, globalParams, executionType string
	var warningGroupID, timeout int
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an empty workflow definition.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			body := map[string]any{
				"name":           args[0],
				"description":    description,
				"projectCode":    projectCode,
				"releaseState":   releaseState,
				"globalParams":   globalParams,
				"warningGroupId": warningGroupID,
				"timeout":        timeout,
				"executionType":  executionType,
			}
			return apiRun(cmd, *flags, "workflow.create", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodPost, "/v2/workflows", body)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().StringVar(&description, "description", "", "Workflow description")
	c.Flags().StringVar(&releaseState, "release-state", "OFFLINE", "Release state: ONLINE or OFFLINE")
	c.Flags().StringVar(&globalParams, "global-params", "[]", "Global params JSON")
	c.Flags().StringVar(&executionType, "execution-type", "PARALLEL", "Execution type")
	c.Flags().IntVar(&warningGroupID, "warning-group-id", 0, "Warning group ID")
	c.Flags().IntVar(&timeout, "timeout", 0, "Workflow timeout minutes")
	return c
}

func newWorkflowUpdateCmd(flags *apiFlags) *cobra.Command {
	var name, description, releaseState, globalParams, executionType, location string
	var warningGroupID, timeout int
	c := &cobra.Command{
		Use:   "update <workflow-code>",
		Short: "Update workflow definition metadata.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			body := map[string]any{}
			if name != "" {
				body["name"] = name
			}
			if description != "" {
				body["description"] = description
			}
			if releaseState != "" {
				body["releaseState"] = releaseState
			}
			if globalParams != "" {
				body["globalParams"] = globalParams
			}
			if executionType != "" {
				body["executionType"] = executionType
			}
			if location != "" {
				body["location"] = location
			}
			if warningGroupID != 0 {
				body["warningGroupId"] = warningGroupID
			}
			if timeout != 0 {
				body["timeout"] = timeout
			}
			return apiRun(cmd, *flags, "workflow.update", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodPut, fmt.Sprintf("/v2/workflows/%d", code), body)
			})
		},
	}
	c.Flags().StringVar(&name, "name", "", "Workflow name")
	c.Flags().StringVar(&description, "description", "", "Workflow description")
	c.Flags().StringVar(&releaseState, "release-state", "", "Release state")
	c.Flags().StringVar(&globalParams, "global-params", "", "Global params JSON")
	c.Flags().StringVar(&executionType, "execution-type", "", "Execution type")
	c.Flags().StringVar(&location, "location", "", "Location JSON")
	c.Flags().IntVar(&warningGroupID, "warning-group-id", 0, "Warning group ID")
	c.Flags().IntVar(&timeout, "timeout", 0, "Workflow timeout minutes")
	return c
}

func newWorkflowGetCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <workflow-code>",
		Short: "Get a workflow definition.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "workflow.get", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodGet, fmt.Sprintf("/v2/workflows/%d", code), nil)
			})
		},
	}
}

func newWorkflowListCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "list",
		Short: "List workflow definitions in a project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			return apiRun(cmd, *flags, "workflow.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet, fmt.Sprintf("/projects/%d/workflow-definition/list", projectCode), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newWorkflowReleaseCmd(flags *apiFlags, name, state string) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   name + " <workflow-code>",
		Short: name + " a workflow definition.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "workflow."+name, func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return releaseWorkflow(ctx, client, projectCode, code, state)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newWorkflowDeleteCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <workflow-code>",
		Short: "Delete a workflow definition.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "workflow.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodDelete, fmt.Sprintf("/v2/workflows/%d", code), nil)
			})
		},
	}
}

func newWorkflowGetDetailCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "get-detail <workflow-code>",
		Short: "Get a workflow definition including task definitions and relations.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "workflow.get-detail", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodGet,
					fmt.Sprintf("/projects/%d/workflow-definition/%d", projectCode, code), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newWorkflowPatchTaskCmd(flags *apiFlags) *cobra.Command {
	var projectCode, taskCode int64
	var rawScript, rawScriptFile string
	var keepOffline bool
	c := &cobra.Command{
		Use:   "patch-task <workflow-code>",
		Short: "Replace one task's rawScript in a workflow definition (offline, update, restore release state).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			if taskCode == 0 {
				return fmt.Errorf("--task-code is required")
			}
			if rawScriptFile != "" {
				b, err := os.ReadFile(rawScriptFile)
				if err != nil {
					return err
				}
				rawScript = string(b)
			}
			if rawScript == "" {
				return fmt.Errorf("--raw-script or --raw-script-file is required")
			}
			workflowCode, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			client, profile, err := apiClient(*flags)
			if err != nil {
				writeAPIError(cmd, "workflow.patch-task", "CONFIG_ERROR", err)
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
			defer cancel()
			result, err := dsapi.PatchWorkflowTask(ctx, client, dsapi.WorkflowPatchInput{
				ProjectCode:  projectCode,
				WorkflowCode: workflowCode,
				TaskCode:     taskCode,
				NewRawScript: rawScript,
				KeepOffline:  keepOffline,
			})
			if err != nil {
				writeAPIError(cmd, "workflow.patch-task", "DS_API_ERROR", err)
				return err
			}
			e := output.NewEnvelope("workflow.patch-task")
			e.Summary = map[string]any{
				"cluster":              profile.Name,
				"api_url":              profile.APIURL,
				"project_code":         projectCode,
				"workflow_code":        workflowCode,
				"task_code":            taskCode,
				"prev_release_state":   result.PrevReleaseState,
				"final_release_state":  result.FinalReleaseState,
				"workflow_name":        result.WorkflowName,
				"new_workflow_version": result.NewWorkflowVersion,
			}
			e.Data = map[string]any{
				"update_response": result.UpdateResponse,
			}
			return e.Write(cmd.OutOrStdout())
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().Int64Var(&taskCode, "task-code", 0, "Task definition code inside the workflow")
	c.Flags().StringVar(&rawScript, "raw-script", "", "New rawScript for the task")
	c.Flags().StringVar(&rawScriptFile, "raw-script-file", "", "Read new rawScript from this file")
	c.Flags().BoolVar(&keepOffline, "keep-offline", false, "Leave the workflow OFFLINE after update even if it was ONLINE before")
	return c
}

func newWorkflowStartCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	var scheduleTime string
	var failureStrategy, warningType, execType, taskDependType, startNodeList string
	var runMode, instancePriority, workerGroup, tenantCode, startParams string
	var complementDependentMode, executionOrder string
	var warningGroupID, expectedParallelism, dryRun int
	var environmentCode int64
	var allLevelDependent bool
	c := &cobra.Command{
		Use:   "start <workflow-code>",
		Short: "Trigger one workflow run via /executors/start-workflow-instance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			code, err := int64Arg(args[0], "workflow-code")
			if err != nil {
				return err
			}
			if scheduleTime == "" {
				scheduleTime = time.Now().UTC().Format("2006-01-02 15:04:05")
			}
			return apiRun(cmd, *flags, "workflow.start", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				values := formValues(
					"workflowDefinitionCode", strconv.FormatInt(code, 10),
					"scheduleTime", scheduleTime,
					"failureStrategy", failureStrategy,
					"warningType", warningType,
					"execType", execType,
					"taskDependType", taskDependType,
					"startNodeList", startNodeList,
					"runMode", runMode,
					"workflowInstancePriority", instancePriority,
					"workerGroup", workerGroup,
					"tenantCode", tenantCode,
					"startParams", startParams,
					"complementDependentMode", complementDependentMode,
					"executionOrder", executionOrder,
				)
				if warningGroupID > 0 {
					values.Set("warningGroupId", strconv.Itoa(warningGroupID))
				}
				if expectedParallelism > 0 {
					values.Set("expectedParallelismNumber", strconv.Itoa(expectedParallelism))
				}
				if dryRun != 0 {
					values.Set("dryRun", strconv.Itoa(dryRun))
				}
				if environmentCode != 0 {
					values.Set("environmentCode", strconv.FormatInt(environmentCode, 10))
				}
				if allLevelDependent {
					values.Set("allLevelDependent", "true")
				}
				return client.Form(ctx, http.MethodPost,
					fmt.Sprintf("/projects/%d/executors/start-workflow-instance", projectCode), values)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().StringVar(&scheduleTime, "schedule-time", "", "Schedule time, e.g. 2026-01-01 00:00:00 (defaults to now UTC)")
	c.Flags().StringVar(&failureStrategy, "failure-strategy", "CONTINUE", "Failure strategy: CONTINUE or END")
	c.Flags().StringVar(&warningType, "warning-type", "NONE", "Warning type: NONE, SUCCESS, FAILURE, ALL")
	c.Flags().StringVar(&execType, "exec-type", "START_PROCESS", "Command type: START_PROCESS or COMPLEMENT_DATA")
	c.Flags().StringVar(&taskDependType, "task-depend-type", "", "Task dependency type: TASK_POST/TASK_PRE/TASK_ONLY")
	c.Flags().StringVar(&startNodeList, "start-node-list", "", "Comma-separated start node codes")
	c.Flags().StringVar(&runMode, "run-mode", "", "Run mode: RUN_MODE_SERIAL or RUN_MODE_PARALLEL")
	c.Flags().StringVar(&instancePriority, "instance-priority", "", "Workflow instance priority: HIGHEST..LOWEST")
	c.Flags().StringVar(&workerGroup, "worker-group", "", "Worker group")
	c.Flags().StringVar(&tenantCode, "tenant-code", "", "Tenant code")
	c.Flags().StringVar(&startParams, "start-params", "", "Start params JSON")
	c.Flags().StringVar(&complementDependentMode, "complement-dependent-mode", "", "Complement dependent mode")
	c.Flags().StringVar(&executionOrder, "execution-order", "", "Execution order")
	c.Flags().IntVar(&warningGroupID, "warning-group-id", 0, "Warning group ID")
	c.Flags().IntVar(&expectedParallelism, "expected-parallelism", 0, "Expected parallelism for complement parallel mode")
	c.Flags().IntVar(&dryRun, "dry-run", 0, "Dry-run flag")
	c.Flags().Int64Var(&environmentCode, "environment-code", 0, "Environment code")
	c.Flags().BoolVar(&allLevelDependent, "all-level-dependent", false, "All-level dependent")
	return c
}
