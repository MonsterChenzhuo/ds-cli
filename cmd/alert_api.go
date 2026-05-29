package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/spf13/cobra"
)

func newAlertCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:   "alert",
		Short: "Manage DolphinScheduler alert groups.",
	}
	addAPIFlags(cmd, &flags)
	group := &cobra.Command{Use: "group", Short: "Manage alert groups"}
	group.AddCommand(newAlertGroupCreateCmd(&flags))
	group.AddCommand(newAlertGroupUpdateCmd(&flags))
	group.AddCommand(newAlertGroupListCmd(&flags))
	group.AddCommand(newAlertGroupDeleteCmd(&flags))
	cmd.AddCommand(group)
	return cmd
}

func newAlertGroupCreateCmd(flags *apiFlags) *cobra.Command {
	var description, instanceIDs string
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an alert group.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRun(cmd, *flags, "alert.group.create", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost, "/alert-groups", formValues(
					"groupName", args[0],
					"description", description,
					"alertInstanceIds", instanceIDs,
				))
			})
		},
	}
	c.Flags().StringVar(&description, "description", "", "Alert group description")
	c.Flags().StringVar(&instanceIDs, "alert-instance-ids", "", "Comma-separated alert plugin instance IDs")
	_ = c.MarkFlagRequired("alert-instance-ids")
	return c
}

func newAlertGroupUpdateCmd(flags *apiFlags) *cobra.Command {
	var name, description, instanceIDs string
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an alert group.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := intArg(args[0], "alert-group-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "alert.group.update", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPut, fmt.Sprintf("/alert-groups/%d", id), formValues(
					"groupName", name,
					"description", description,
					"alertInstanceIds", instanceIDs,
				))
			})
		},
	}
	c.Flags().StringVar(&name, "name", "", "Alert group name")
	c.Flags().StringVar(&description, "description", "", "Alert group description")
	c.Flags().StringVar(&instanceIDs, "alert-instance-ids", "", "Comma-separated alert plugin instance IDs")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("alert-instance-ids")
	return c
}

func newAlertGroupListCmd(flags *apiFlags) *cobra.Command {
	var pageNo, pageSize int
	var search string
	c := &cobra.Command{
		Use:   "list",
		Short: "List alert groups.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRun(cmd, *flags, "alert.group.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet, "/alert-groups", formValues(
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

func newAlertGroupDeleteCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an alert group.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := intArg(args[0], "alert-group-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "alert.group.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodDelete, fmt.Sprintf("/alert-groups/%d", id), nil)
			})
		},
	}
}
