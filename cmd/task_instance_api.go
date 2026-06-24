package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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
	var full, clean bool
	var outputPath string
	c := &cobra.Command{
		Use:   "log <task-instance-id>",
		Short: "Fetch a task instance's log (paged slice, or full with --full).",
		Long: "log fetches a task instance's log via /log/detail.\n\n" +
			"By default it returns a single page of up to --limit lines starting at --skip-line-num.\n" +
			"--full loops over the pages until the whole log is fetched, so an agent never\n" +
			"silently truncates a long log. --output FILE writes the log to disk and prints only\n" +
			"a JSON envelope summary (path/bytes/lines). --clean strips the DolphinScheduler\n" +
			"per-line prefix (timestamp + 'INFO  -  -> ') to leave readable text; off by default.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := intArg(args[0], "task-instance-id")
			if err != nil {
				return err
			}
			// Single-page mode without output/clean keeps the original envelope passthrough.
			if !full && outputPath == "" && !clean {
				return apiRun(cmd, *flags, "task-instance.log", func(ctx context.Context, client *dsapi.Client) (*dsapi.Response, error) {
					return client.Form(ctx, http.MethodGet, "/log/detail",
						formValues(
							"taskInstanceId", strconv.Itoa(id),
							"skipLineNum", strconv.Itoa(skipLineNum),
							"limit", strconv.Itoa(limit),
						))
				})
			}

			client, profile, err := apiClient(*flags)
			if err != nil {
				writeAPIError(cmd, "task-instance.log", "CONFIG_ERROR", err)
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), profile.Timeout)
			defer cancel()

			message, lines, err := fetchTaskLog(ctx, client, id, skipLineNum, limit, full)
			if err != nil {
				writeAPIError(cmd, "task-instance.log", "DS_API_ERROR", err)
				return err
			}
			if clean {
				message = cleanDSLog(message)
			}

			if outputPath != "" {
				if err := os.WriteFile(outputPath, []byte(message), 0o644); err != nil {
					writeAPIError(cmd, "task-instance.log", "IO_ERROR", err)
					return err
				}
				e := output.NewEnvelope("task-instance.log")
				e.Summary = map[string]any{"cluster": profile.Name, "api_url": profile.APIURL}
				e.Data = map[string]any{
					"output_path": outputPath,
					"bytes":       len(message),
					"lines":       lines,
					"full":        full,
					"clean":       clean,
				}
				return e.Write(cmd.OutOrStdout())
			}

			e := output.NewEnvelope("task-instance.log")
			e.Summary = map[string]any{"cluster": profile.Name, "api_url": profile.APIURL}
			e.Data = map[string]any{"message": message, "lines": lines, "full": full, "clean": clean}
			return e.Write(cmd.OutOrStdout())
		},
	}
	c.Flags().IntVar(&skipLineNum, "skip-line-num", 0, "Lines to skip from the top of the log")
	c.Flags().IntVar(&limit, "limit", 1000, "Max lines per request")
	c.Flags().BoolVar(&full, "full", false, "Fetch the whole log by paging until exhausted")
	c.Flags().StringVar(&outputPath, "output", "", "Write the log to this file; stdout then gets only an envelope summary")
	c.Flags().BoolVar(&clean, "clean", false, "Strip the DolphinScheduler per-line prefix to leave readable text")
	return c
}

// taskLogPage is the data payload of /log/detail.
type taskLogPage struct {
	LineNum int    `json:"lineNum"`
	Message string `json:"message"`
}

// fetchTaskLog reads one page (full=false) or loops until the log is exhausted
// (full=true). It returns the concatenated message and the total line count.
// A page returning fewer than `limit` lines marks the end of the log.
func fetchTaskLog(ctx context.Context, client *dsapi.Client, id, skip, limit int, full bool) (string, int, error) {
	var buf strings.Builder
	total := 0
	for {
		resp, err := client.Form(ctx, http.MethodGet, "/log/detail",
			formValues(
				"taskInstanceId", strconv.Itoa(id),
				"skipLineNum", strconv.Itoa(skip),
				"limit", strconv.Itoa(limit),
			))
		if err != nil {
			return "", 0, err
		}
		var decoded struct {
			Data taskLogPage `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &decoded); err != nil {
			return "", 0, fmt.Errorf("decode log page: %w", err)
		}
		page := decoded.Data
		buf.WriteString(page.Message)
		total += page.LineNum
		if !full || page.LineNum < limit || page.LineNum == 0 {
			break
		}
		skip += page.LineNum
	}
	return buf.String(), total, nil
}

// dsLogPrefix matches the DolphinScheduler per-line log prefix, e.g.
// "2026-06-24 03:00:16.421 INFO  - " optionally followed by " -> " for
// nested process output.
var dsLogPrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?\s+\w+\s+-\s+(?:-> )?`)

// cleanDSLog strips the DS prefix from every line, leaving the raw log content.
func cleanDSLog(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		trimmed := strings.TrimRight(ln, "\r")
		lines[i] = dsLogPrefix.ReplaceAllString(trimmed, "")
	}
	return strings.Join(lines, "\n")
}

func newTaskInstanceLogDownloadCmd(flags *apiFlags) *cobra.Command {
	var outputPath string
	c := &cobra.Command{
		Use:   "log-download <task-instance-id>",
		Short: "Download a task instance's whole log to a local file.",
		Long: "log-download fetches the whole log via /log/download-log (no line limit) and writes\n" +
			"it to a local file. With --output FILE it writes there; without --output it writes\n" +
			"<tmpdir>/ds-cli/<task-instance-id>.log. Either way stdout is only a JSON envelope\n" +
			"summary (output_path/bytes) — the raw bytes are never written to stdout, so an agent\n" +
			"can download the full log and then read it from disk.",
		Args: cobra.ExactArgs(1),
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
			// download-log returns binary on success but a JSON error body on failure.
			// Guard against silently writing a JSON error blob to a .log file.
			if looksLikeJSONError(resp.Body) {
				writeAPIError(cmd, "task-instance.log-download", "DS_API_ERROR",
					fmt.Errorf("download-log returned a JSON error instead of log bytes: %s", trimForError(resp.Body)))
				return fmt.Errorf("download-log failed")
			}

			target := outputPath
			if target == "" {
				dir := filepath.Join(os.TempDir(), "ds-cli")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					writeAPIError(cmd, "task-instance.log-download", "IO_ERROR", err)
					return err
				}
				target = filepath.Join(dir, strconv.Itoa(id)+".log")
			}
			if err := os.WriteFile(target, resp.Body, 0o644); err != nil {
				writeAPIError(cmd, "task-instance.log-download", "IO_ERROR", err)
				return err
			}
			abs, _ := filepath.Abs(target)
			e := output.NewEnvelope("task-instance.log-download")
			e.Summary = map[string]any{
				"cluster":     profile.Name,
				"api_url":     profile.APIURL,
				"http_status": resp.HTTPStatus,
			}
			e.Data = map[string]any{
				"output_path": abs,
				"bytes":       len(resp.Body),
			}
			return e.Write(cmd.OutOrStdout())
		},
	}
	c.Flags().StringVar(&outputPath, "output", "", "Write the log to this file (default: <tmpdir>/ds-cli/<id>.log)")
	return c
}

// looksLikeJSONError reports whether body is a DS JSON error envelope rather
// than raw log bytes. download-log returns octet-stream on success and a
// {"code":...,"msg":...} body on failure.
func looksLikeJSONError(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	var probe struct {
		Code *int   `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil {
		return false
	}
	return probe.Code != nil && *probe.Code != 0
}

// trimForError returns a short single-line preview of a body for error messages.
func trimForError(body []byte) string {
	s := strings.TrimSpace(string(body))
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}
