package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/spf13/cobra"
)

func newScheduleCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage DolphinScheduler schedules through the REST API.",
	}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newScheduleCreateCmd(&flags))
	cmd.AddCommand(newScheduleUpdateCmd(&flags))
	cmd.AddCommand(newScheduleGetCmd(&flags))
	cmd.AddCommand(newScheduleListCmd(&flags))
	cmd.AddCommand(newScheduleStateCmd(&flags, "online"))
	cmd.AddCommand(newScheduleStateCmd(&flags, "offline"))
	cmd.AddCommand(newScheduleDeleteCmd(&flags))
	return cmd
}

type scheduleFields struct {
	WorkflowCode int64
	Crontab      string
	StartTime    string
	EndTime      string
	Timezone     string
	ReleaseState string
	Failure      string
	WarningType  string
	WarningGroup int
	Priority     string
	WorkerGroup  string
	TenantCode   string
	EnvCode      int64
}

func bindScheduleFlags(c *cobra.Command, f *scheduleFields, includeWorkflow bool) {
	if includeWorkflow {
		c.Flags().Int64Var(&f.WorkflowCode, "workflow-code", 0, "Workflow definition code")
	}
	c.Flags().StringVar(&f.Crontab, "crontab", "", "Quartz cron expression")
	c.Flags().StringVar(&f.StartTime, "start-time", "", "Schedule start time, e.g. 2026-01-01 00:00:00")
	c.Flags().StringVar(&f.EndTime, "end-time", "", "Schedule end time, e.g. 2099-01-01 00:00:00")
	c.Flags().StringVar(&f.Timezone, "timezone", "Asia/Shanghai", "Timezone ID")
	c.Flags().StringVar(&f.ReleaseState, "release-state", "OFFLINE", "Release state: ONLINE or OFFLINE")
	c.Flags().StringVar(&f.Failure, "failure-strategy", "CONTINUE", "Failure strategy: CONTINUE or END")
	c.Flags().StringVar(&f.WarningType, "warning-type", "NONE", "Warning type: NONE, SUCCESS, FAILURE, ALL")
	c.Flags().IntVar(&f.WarningGroup, "warning-group-id", 0, "Warning group ID")
	c.Flags().StringVar(&f.Priority, "priority", "MEDIUM", "Workflow instance priority")
	c.Flags().StringVar(&f.WorkerGroup, "worker-group", "default", "Worker group")
	c.Flags().StringVar(&f.TenantCode, "tenant-code", "default", "Tenant code")
	c.Flags().Int64Var(&f.EnvCode, "environment-code", 0, "Environment code")
}

func (f scheduleFields) body() (map[string]any, error) {
	if f.WorkflowCode == 0 {
		return nil, fmt.Errorf("--workflow-code is required")
	}
	if f.Crontab == "" || f.StartTime == "" || f.EndTime == "" {
		return nil, fmt.Errorf("--crontab, --start-time, and --end-time are required")
	}
	return map[string]any{
		"workflowDefinitionCode":   f.WorkflowCode,
		"crontab":                  f.Crontab,
		"startTime":                f.StartTime,
		"endTime":                  f.EndTime,
		"timezoneId":               f.Timezone,
		"releaseState":             f.ReleaseState,
		"failureStrategy":          f.Failure,
		"warningType":              f.WarningType,
		"warningGroupId":           f.WarningGroup,
		"workflowInstancePriority": f.Priority,
		"workerGroup":              f.WorkerGroup,
		"tenantCode":               f.TenantCode,
		"environmentCode":          f.EnvCode,
	}, nil
}

func newScheduleCreateCmd(flags *apiFlags) *cobra.Command {
	var fields scheduleFields
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a schedule.",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := fields.body()
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "schedule.create", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodPost, "/v2/schedules", body)
			})
		},
	}
	bindScheduleFlags(c, &fields, true)
	return c
}

func newScheduleUpdateCmd(flags *apiFlags) *cobra.Command {
	var fields scheduleFields
	c := &cobra.Command{
		Use:   "update <schedule-id>",
		Short: "Update a schedule.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := intArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			body, err := fields.body()
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "schedule.update", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodPut, fmt.Sprintf("/v2/schedules/%d", id), body)
			})
		},
	}
	bindScheduleFlags(c, &fields, true)
	return c
}

func newScheduleGetCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <schedule-id>",
		Short: "Get a schedule.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := intArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "schedule.get", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodGet, fmt.Sprintf("/v2/schedules/%d", id), nil)
			})
		},
	}
}

func newScheduleListCmd(flags *apiFlags) *cobra.Command {
	var projectCode, workflowCode int64
	var search string
	var pageNo, pageSize int
	c := &cobra.Command{
		Use:   "list",
		Short: "List schedules in a project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			return apiRun(cmd, *flags, "schedule.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet, fmt.Sprintf("/projects/%d/schedules", projectCode), formValues(
					"workflowDefinitionCode", strconv.FormatInt(workflowCode, 10),
					"searchVal", search,
					"pageNo", strconv.Itoa(pageNo),
					"pageSize", strconv.Itoa(pageSize),
				))
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().Int64Var(&workflowCode, "workflow-code", 0, "Workflow definition code")
	c.Flags().StringVar(&search, "search", "", "Search text")
	c.Flags().IntVar(&pageNo, "page-no", 1, "Page number")
	c.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return c
}

func newScheduleStateCmd(flags *apiFlags, state string) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   state + " <schedule-id>",
		Short: state + " a schedule.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "schedule."+state, func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost, fmt.Sprintf("/projects/%d/schedules/%d/%s", projectCode, id, state), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newScheduleDeleteCmd(flags *apiFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <schedule-id>",
		Short: "Delete a schedule.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := intArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "schedule.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodDelete, fmt.Sprintf("/v2/schedules/%d", id), nil)
			})
		},
	}
}
