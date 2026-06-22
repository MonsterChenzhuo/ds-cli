package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/spf13/cobra"
)

func newTaskDefCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:   "task-def",
		Short: "Read and update individual task definitions by code.",
	}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newTaskDefGetCmd(&flags))
	cmd.AddCommand(newTaskDefUpdateCmd(&flags))
	return cmd
}

func newTaskDefGetCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "get <task-code>",
		Short: "Get a task definition by code.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			code, err := int64Arg(args[0], "task-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "task-def.get", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodGet,
					fmt.Sprintf("/projects/%d/task-definition/%d", projectCode, code), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newTaskDefUpdateCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	var rawScript, rawScriptFile, upstreamCodes string
	c := &cobra.Command{
		Use:   "update <task-code>",
		Short: "Replace a task definition's rawScript via /task-definition/{code}/with-upstream.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
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
			code, err := int64Arg(args[0], "task-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "task-def.update", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return dsapi.PatchTaskDefinition(ctx, client, projectCode, code, rawScript, upstreamCodes)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().StringVar(&rawScript, "raw-script", "", "New rawScript for the task")
	c.Flags().StringVar(&rawScriptFile, "raw-script-file", "", "Read new rawScript from this file")
	c.Flags().StringVar(&upstreamCodes, "upstream-codes", "", "Comma-separated upstream task codes (optional)")
	return c
}
