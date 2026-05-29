package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ds-cli/ds-cli/internal/dsapi"
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
	cmd.AddCommand(newWorkflowListCmd(&flags))
	cmd.AddCommand(newWorkflowReleaseCmd(&flags, "online", "ONLINE"))
	cmd.AddCommand(newWorkflowReleaseCmd(&flags, "offline", "OFFLINE"))
	cmd.AddCommand(newWorkflowDeleteCmd(&flags))
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
