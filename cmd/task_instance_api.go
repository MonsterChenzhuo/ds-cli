package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/spf13/cobra"
)

func newTaskInstanceCmd() *cobra.Command {
	var flags apiFlags
	cmd := &cobra.Command{
		Use:     "task-instance",
		Aliases: []string{"ti"},
		Short:   "Inspect and operate DolphinScheduler task instances.",
	}
	addAPIFlags(cmd, &flags)
	cmd.AddCommand(newTaskInstanceListCmd(&flags))
	cmd.AddCommand(newTaskInstanceForceSuccessCmd(&flags))
	cmd.AddCommand(newTaskInstanceStopCmd(&flags))
	cmd.AddCommand(newTaskInstanceLogCmd(&flags))
	cmd.AddCommand(newTaskInstanceLogDownloadCmd(&flags))
	return cmd
}

func newTaskInstanceListCmd(flags *apiFlags) *cobra.Command {
	var projectCode, taskCode int64
	var workflowInstanceID int
	var workflowInstanceName, workflowDefinitionName, taskName, executorName string
	var stateType, host, startDate, endDate, search, taskExecuteType string
	var pageNo, pageSize int
	c := &cobra.Command{
		Use:   "list",
		Short: "List task instances in a project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			return apiRun(cmd, *flags, "task-instance.list", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				values := formValues(
					"searchVal", search,
					"workflowInstanceName", workflowInstanceName,
					"workflowDefinitionName", workflowDefinitionName,
					"taskName", taskName,
					"executorName", executorName,
					"stateType", stateType,
					"host", host,
					"startDate", startDate,
					"endDate", endDate,
					"taskExecuteType", taskExecuteType,
					"pageNo", strconv.Itoa(pageNo),
					"pageSize", strconv.Itoa(pageSize),
				)
				if workflowInstanceID > 0 {
					values.Set("workflowInstanceId", strconv.Itoa(workflowInstanceID))
				}
				if taskCode != 0 {
					values.Set("taskCode", strconv.FormatInt(taskCode, 10))
				}
				return client.Form(ctx, http.MethodGet,
					fmt.Sprintf("/projects/%d/task-instances", projectCode), values)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	c.Flags().IntVar(&workflowInstanceID, "workflow-instance-id", 0, "Filter by workflow instance id")
	c.Flags().Int64Var(&taskCode, "task-code", 0, "Filter by task definition code")
	c.Flags().StringVar(&workflowInstanceName, "workflow-instance-name", "", "Filter by workflow instance name")
	c.Flags().StringVar(&workflowDefinitionName, "workflow-definition-name", "", "Filter by workflow definition name")
	c.Flags().StringVar(&taskName, "task-name", "", "Filter by task name")
	c.Flags().StringVar(&executorName, "executor-name", "", "Filter by executor user name")
	c.Flags().StringVar(&stateType, "state-type", "", "Filter by task execution status")
	c.Flags().StringVar(&host, "host", "", "Filter by worker host")
	c.Flags().StringVar(&startDate, "start-date", "", "Filter range start")
	c.Flags().StringVar(&endDate, "end-date", "", "Filter range end")
	c.Flags().StringVar(&search, "search", "", "Search text")
	c.Flags().StringVar(&taskExecuteType, "task-execute-type", "", "Task execute type: BATCH or STREAM")
	c.Flags().IntVar(&pageNo, "page-no", 1, "Page number")
	c.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return c
}

func newTaskInstanceForceSuccessCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "force-success <task-instance-id>",
		Short: "Mark a failed task instance as forced success.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "task-instance-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "task-instance.force-success", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost,
					fmt.Sprintf("/projects/%d/task-instances/%d/force-success", projectCode, id), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newTaskInstanceStopCmd(flags *apiFlags) *cobra.Command {
	var projectCode int64
	c := &cobra.Command{
		Use:   "stop <task-instance-id>",
		Short: "Stop a stream task instance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectCode == 0 {
				return fmt.Errorf("--project-code is required")
			}
			id, err := intArg(args[0], "task-instance-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "task-instance.stop", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodPost,
					fmt.Sprintf("/projects/%d/task-instances/%d/stop", projectCode, id), nil)
			})
		},
	}
	c.Flags().Int64Var(&projectCode, "project-code", 0, "Project code")
	return c
}

func newTaskInstanceLogCmd(flags *apiFlags) *cobra.Command {
	var skipLineNum, limit int
	c := &cobra.Command{
		Use:   "log <task-instance-id>",
		Short: "Fetch a paged slice of a task instance's log.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := intArg(args[0], "task-instance-id")
			if err != nil {
				return err
			}
			return apiRun(cmd, *flags, "task-instance.log", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
				return client.Form(ctx, http.MethodGet, "/log/detail",
					formValues(
						"taskInstanceId", strconv.Itoa(id),
						"skipLineNum", strconv.Itoa(skipLineNum),
						"limit", strconv.Itoa(limit),
					))
			})
		},
	}
	c.Flags().IntVar(&skipLineNum, "skip-line-num", 0, "Lines to skip from the top of the log")
	c.Flags().IntVar(&limit, "limit", 1000, "Max lines to return")
	return c
}

func newTaskInstanceLogDownloadCmd(flags *apiFlags) *cobra.Command {
	var outputPath string
	c := &cobra.Command{
		Use:   "log-download <task-instance-id>",
		Short: "Download a task instance's full log (binary stream).",
		Long:  "log-download streams the raw attachment returned by /log/download-log. With --output FILE the bytes are written to disk and stdout gets a JSON envelope summary; without --output the bytes are written to stdout (no envelope).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := intArg(args[0], "task-instance-id")
			if err != nil {
				return err
			}
			client, profile, err := apiClient(*flags)
			if err != nil {
				writeAPIError(cmd, "task-instance.log-download", "CONFIG_ERROR", err)
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
			defer cancel()
			resp, err := client.RawGet(ctx, "/log/download-log",
				formValues("taskInstanceId", strconv.Itoa(id)))
			if err != nil {
				writeAPIError(cmd, "task-instance.log-download", "DS_API_ERROR", err)
				return err
			}
			if outputPath == "" {
				_, err := cmd.OutOrStdout().Write(resp.Body)
				return err
			}
			if err := os.WriteFile(outputPath, resp.Body, 0o644); err != nil {
				writeAPIError(cmd, "task-instance.log-download", "DS_API_ERROR", err)
				return err
			}
			e := output.NewEnvelope("task-instance.log-download")
			e.Summary = map[string]any{
				"cluster":     profile.Name,
				"api_url":     profile.APIURL,
				"http_status": resp.HTTPStatus,
			}
			e.Data = map[string]any{
				"output_path": outputPath,
				"bytes":       len(resp.Body),
			}
			return e.Write(cmd.OutOrStdout())
		},
	}
	c.Flags().StringVar(&outputPath, "output", "", "Write the log to this file; otherwise stream bytes to stdout")
	return c
}
