package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/spf13/cobra"
)

func newProjectCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage DolphinScheduler projects through the REST API.",
	}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newProjectCreateCmd(&flags))
	cmd.AddCommand(newProjectListCmd(&flags))
	cmd.AddCommand(newProjectGetCmd(&flags))
	cmd.AddCommand(newProjectDeleteCmd(&flags))
	return cmd
}

func newProjectCreateCmd(flags *apiFlags) *cobra.Command {
	var description string
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a DolphinScheduler project.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"projectName": args[0], "description": description}
			return apiRun(cmd, *flags, "project.create", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodPost, "/v2/projects", body)
			})
		},
	}
	c.Flags().StringVar(&description, "description", "", "Project description")
	return c
}

func newProjectListCmd(flags *apiFlags) *cobra.Command {
	var searchVal string
	var pageNo, pageSize int
	c := &cobra.Command{
		Use:   "list",
		Short: "List DolphinScheduler projects.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRun(cmd, *flags, "project.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet, "/projects", formValues(
					"searchVal", searchVal,
					"pageNo", strconv.Itoa(pageNo),
					"pageSize", strconv.Itoa(pageSize),
				))
			})
		},
	}
	c.Flags().StringVar(&searchVal, "search", "", "Search text")
	c.Flags().IntVar(&pageNo, "page-no", 1, "Page number")
	c.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return c
}

func newProjectGetCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <project-code>",
		Short: "Get a DolphinScheduler project by code.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "project-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "project.get", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodGet, fmt.Sprintf("/v2/projects/%d", code), nil)
			})
		},
	}
}

func newProjectDeleteCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <project-code>",
		Short: "Delete a DolphinScheduler project by code.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "project-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "project.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodDelete, fmt.Sprintf("/v2/projects/%d", code), nil)
			})
		},
	}
}
