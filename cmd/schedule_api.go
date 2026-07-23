package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/ds-cli/ds-cli/internal/output"
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
	ProjectCode  int64
	WorkflowCode int64
	Crontab      string
	StartTime    string
	EndTime      string
	Timezone     string
	Failure      string
	WarningType  string
	WarningGroup int
	Priority     string
	WorkerGroup  string
	TenantCode   string
	EnvCode      int64
}

func bindScheduleFlags(c *cobra.Command, f *scheduleFields, includeWorkflow bool) {
	c.Flags().Int64Var(&f.ProjectCode, "project-code", 0, "Project code (required)")
	if includeWorkflow {
		c.Flags().Int64Var(&f.WorkflowCode, "workflow-code", 0, "Workflow definition code")
	}
	c.Flags().StringVar(&f.Crontab, "crontab", "", "Quartz cron expression")
	c.Flags().StringVar(&f.StartTime, "start-time", "", "Schedule start time, e.g. 2026-01-01 00:00:00")
	c.Flags().StringVar(&f.EndTime, "end-time", "", "Schedule end time, e.g. 2099-01-01 00:00:00")
	c.Flags().StringVar(&f.Timezone, "timezone", "UTC", "Timezone ID")
	c.Flags().StringVar(&f.Failure, "failure-strategy", "CONTINUE", "Failure strategy: CONTINUE or END")
	c.Flags().StringVar(&f.WarningType, "warning-type", "NONE", "Warning type: NONE, SUCCESS, FAILURE, ALL")
	c.Flags().IntVar(&f.WarningGroup, "warning-group-id", 0, "Warning group ID")
	c.Flags().StringVar(&f.Priority, "priority", "MEDIUM", "Workflow instance priority")
	c.Flags().StringVar(&f.WorkerGroup, "worker-group", "default", "Worker group")
	c.Flags().StringVar(&f.TenantCode, "tenant-code", "default", "Tenant code")
	c.Flags().Int64Var(&f.EnvCode, "environment-code", 0, "Environment code")
}

// scheduleJSON packs the timing fields into the single `schedule` string
// parameter the DS 3.4.1 legacy endpoint expects, e.g.
// {"startTime":"..","endTime":"..","crontab":"..","timezoneId":".."}.
// Built via json.Marshal so values are always properly escaped.
func (f scheduleFields) scheduleJSON() (string, error) {
	b, err := json.Marshal(map[string]string{
		"startTime":  f.StartTime,
		"endTime":    f.EndTime,
		"crontab":    f.Crontab,
		"timezoneId": f.Timezone,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// form builds the legacy /projects/{projectCode}/schedules form. When
// includeWorkflow is true (create), workflowDefinitionCode is required and added.
func (f scheduleFields) form(includeWorkflow bool) (url.Values, error) {
	if f.ProjectCode == 0 {
		return nil, fmt.Errorf("--project-code is required")
	}
	if includeWorkflow && f.WorkflowCode == 0 {
		return nil, fmt.Errorf("--workflow-code is required")
	}
	if f.Crontab == "" || f.StartTime == "" || f.EndTime == "" {
		return nil, fmt.Errorf("--crontab, --start-time, and --end-time are required")
	}
	if f.EnvCode == 0 {
		return nil, fmt.Errorf("--environment-code is required (run `ds-cli environment list` to find an existing code)")
	}
	sched, err := f.scheduleJSON()
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	if includeWorkflow {
		v.Set("workflowDefinitionCode", strconv.FormatInt(f.WorkflowCode, 10))
	}
	v.Set("schedule", sched)
	v.Set("warningType", f.WarningType)
	v.Set("warningGroupId", strconv.Itoa(f.WarningGroup))
	v.Set("failureStrategy", f.Failure)
	v.Set("workerGroup", f.WorkerGroup)
	v.Set("tenantCode", f.TenantCode)
	v.Set("environmentCode", strconv.FormatInt(f.EnvCode, 10))
	v.Set("workflowInstancePriority", f.Priority)
	return v, nil
}

func newScheduleCreateCmd(flags *apiFlags) *cobra.Command {
	var fields scheduleFields
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a schedule.",
		RunE: func(cmd *cobra.Command, args []string) error {
			form, err := fields.form(true)
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "schedule.create", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost,
					fmt.Sprintf("/projects/%d/schedules", fields.ProjectCode), form)
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
			// update does not take workflowDefinitionCode (schedule already bound).
			form, err := fields.form(false)
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "schedule.update", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPut,
					fmt.Sprintf("/projects/%d/schedules/%d", fields.ProjectCode, id), form)
			})
		},
	}
	bindScheduleFlags(c, &fields, false)
	return c
}

func newScheduleGetCmd(flags *apiFlags) *cobra.Command {
	var projectCode, workflowCode int64
	c := &cobra.Command{
		Use:   "get <schedule-id>",
		Short: "Get a schedule by id.",
		Long: "DS 3.4.1 has no single-schedule endpoint, so this queries the project's paginated\n" +
			"schedule list and filters locally by id. Pass --workflow-code to narrow the search.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			client, profile, err := apiClient(*flags)
			if err != nil {
				writeAPIError(cmd, "schedule.get", "CONFIG_ERROR", err)
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
			defer cancel()
			item, err := findScheduleByID(ctx, client, projectCode, workflowCode, id)
			if err != nil {
				writeAPIError(cmd, "schedule.get", "DS_API_ERROR", err)
				return err
			}
			e := output.NewEnvelope("schedule.get")
			e.Summary = map[string]any{"cluster": profile.Name, "api_url": profile.APIURL, "project_code": projectCode, "schedule_id": id}
			e.Data = item
			return e.Write(cmd.OutOrStdout())
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code (required)")
	c.Flags().Int64Var(&workflowCode, "workflow-code", 0, "Workflow definition code to narrow the search (optional)")
	return c
}

// findScheduleByID pages through /projects/{pc}/schedules and returns the entry
// whose id matches. DS 3.4.1 exposes no single-schedule GET endpoint.
func findScheduleByID(ctx context.Context, client *dsapi.Client, projectCode, workflowCode int64, id int) (map[string]any, error) {
	const pageSize = 100
	for pageNo := 1; ; pageNo++ {
		resp, err := client.Form(ctx, http.MethodGet,
			fmt.Sprintf("/projects/%d/schedules", projectCode), formValues(
				"workflowDefinitionCode", strconv.FormatInt(workflowCode, 10),
				"pageNo", strconv.Itoa(pageNo),
				"pageSize", strconv.Itoa(pageSize),
			))
		if err != nil {
			return nil, err
		}
		var decoded struct {
			Data struct {
				TotalList []map[string]any `json:"totalList"`
				Total     int              `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &decoded); err != nil {
			return nil, fmt.Errorf("decode schedule list: %w", err)
		}
		for _, s := range decoded.Data.TotalList {
			if int(asInt64Schedule(s["id"])) == id {
				return s, nil
			}
		}
		if len(decoded.Data.TotalList) < pageSize {
			return nil, fmt.Errorf("schedule id %d not found in project %d", id, projectCode)
		}
	}
}

// asInt64Schedule coerces a JSON number/string id into int64.
func asInt64Schedule(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
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
	var projectCode int64
	c := &cobra.Command{
		Use:   "delete <schedule-id>",
		Short: "Delete a schedule.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "schedule.delete", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.JSON(ctx, http.MethodDelete,
					fmt.Sprintf("/projects/%d/schedules/%d", projectCode, id), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code (required)")
	return c
}
