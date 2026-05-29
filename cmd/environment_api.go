package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/spf13/cobra"
)

func newEnvironmentCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env"},
		Short:   "Manage DolphinScheduler task runtime environments.",
	}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newEnvironmentCreateCmd(&flags))
	cmd.AddCommand(newEnvironmentUpdateCmd(&flags))
	cmd.AddCommand(newEnvironmentListCmd(&flags))
	cmd.AddCommand(newEnvironmentGetCmd(&flags))
	cmd.AddCommand(newEnvironmentDeleteCmd(&flags))
	return cmd
}

func newEnvironmentCreateCmd(flags *apiFlags) *cobra.Command {
	var config, description, workerGroups string
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a DolphinScheduler environment.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("--env-config is required")
			}
			return apiRun(cmd, *flags, "environment.create", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost, "/environment/create", formValues(
					"name", args[0],
					"config", config,
					"description", description,
					"workerGroups", workerGroups,
				))
			})
		},
	}
	c.Flags().StringVar(&config, "env-config", "", "Environment exports, e.g. 'export PYTHON_LAUNCHER=/usr/bin/python3'")
	c.Flags().StringVar(&description, "description", "", "Environment description")
	c.Flags().StringVar(&workerGroups, "worker-groups", "", "Comma-separated worker groups")
	return c
}

func newEnvironmentUpdateCmd(flags *apiFlags) *cobra.Command {
	var name, config, description, workerGroups string
	c := &cobra.Command{
		Use:   "update <environment-code>",
		Short: "Update a DolphinScheduler environment.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "environment-code")
			if err != nil {
				return err
			}
			if name == "" || config == "" {
				return fmt.Errorf("--name and --env-config are required")
			}
			return apiRun(cmd, *flags, "environment.update", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost, "/environment/update", formValues(
					"code", strconv.FormatInt(code, 10),
					"name", name,
					"config", config,
					"description", description,
					"workerGroups", workerGroups,
				))
			})
		},
	}
	c.Flags().StringVar(&name, "name", "", "Environment name")
	c.Flags().StringVar(&config, "env-config", "", "Environment exports")
	c.Flags().StringVar(&description, "description", "", "Environment description")
	c.Flags().StringVar(&workerGroups, "worker-groups", "", "Comma-separated worker groups")
	return c
}

func newEnvironmentListCmd(flags *apiFlags) *cobra.Command {
	var pageNo, pageSize int
	var search string
	c := &cobra.Command{
		Use:   "list",
		Short: "List DolphinScheduler environments.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRun(cmd, *flags, "environment.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet, "/environment/list-paging", formValues(
					"searchVal", search,
					"pageNo", strconv.Itoa(pageNo),
					"pageSize", strconv.Itoa(pageSize),
				))
			})
		},
	}
	c.Flags().StringVar(&search, "search", "", "Search text")
	c.Flags().IntVar(&pageNo, "page-no", 1, "Page number")
	c.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return c
}

func newEnvironmentGetCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <environment-code>",
		Short: "Get an environment by code.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "environment-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "environment.get", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet, "/environment/query-by-code", formValues("environmentCode", strconv.FormatInt(code, 10)))
			})
		},
	}
}

func newEnvironmentDeleteCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <environment-code>",
		Short: "Delete an environment by code.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := int64Arg(args[0], "environment-code")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "environment.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost, "/environment/delete", formValues("environmentCode", strconv.FormatInt(code, 10)))
			})
		},
	}
}
