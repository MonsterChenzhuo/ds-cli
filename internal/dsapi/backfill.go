package dsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// WorkflowInstanceSummary is the subset of a DolphinScheduler workflow instance
// that ds-cli needs for listing and backfill-coverage checks. It deliberately
// drops the very large fields (taskDefinitionList/globalParams/locations/...)
// that make the raw API response unreadable for agents.
type WorkflowInstanceSummary struct {
	ID                        int    `json:"id"`
	Name                      string `json:"name"`
	State                     string `json:"state"`
	ScheduleTime              string `json:"scheduleTime"`
	StartTime                 string `json:"startTime"`
	EndTime                   string `json:"endTime"`
	Duration                  string `json:"duration"`
	CommandType               string `json:"commandType"`
	CmdTypeIfComplement       string `json:"cmdTypeIfComplement"`
	CommandParam              string `json:"commandParam,omitempty"`
	WorkflowDefinitionCode    int64  `json:"workflowDefinitionCode"`
	WorkflowDefinitionVersion int    `json:"workflowDefinitionVersion"`
	ExecutorName              string `json:"executorName"`
	Host                      string `json:"host"`
	WorkerGroup               string `json:"workerGroup"`
	TenantCode                string `json:"tenantCode"`
	EnvironmentCode           int64  `json:"environmentCode"`
	FailureStrategy           string `json:"failureStrategy"`
	WarningType               string `json:"warningType"`
	TaskDependType            string `json:"taskDependType"`
	WorkflowInstancePriority  string `json:"workflowInstancePriority"`
}

// WorkflowInstanceListOptions contains the filters supported by the legacy
// workflow-instance list endpoint.
type WorkflowInstanceListOptions struct {
	WorkflowCode int64
	StateType    string
	StartDate    string
	EndDate      string
	ExecutorName string
	Search       string
	Host         string
}

// IsComplement reports whether the instance was created by a "补数" (backfill)
// command.
func (w WorkflowInstanceSummary) IsComplement() bool {
	return w.CmdTypeIfComplement == "COMPLEMENT_DATA" || w.CommandType == "COMPLEMENT_DATA"
}

// ReferencedDates returns the distinct schedule dates (yyyy-MM-dd) this
// instance carries in commandParam.backfillTimeList. In DS a backfill request
// stores the remaining date list on each successor instance, so this is the
// request's *intent*, not what the instance ran (that is ScheduleTime).
func (w WorkflowInstanceSummary) ReferencedDates() []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(raw string) {
		d := NormalizeDate(raw)
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		out = append(out, d)
	}
	if p := parseBackfillTimeList(w.CommandParam); len(p) > 0 {
		for _, d := range p {
			add(d)
		}
	}
	sort.Strings(out)
	return out
}

func parseBackfillTimeList(commandParam string) []string {
	trimmed := strings.TrimSpace(commandParam)
	if trimmed == "" {
		return nil
	}
	var probe struct {
		BackfillTimeList []string `json:"backfillTimeList"`
	}
	if err := json.Unmarshal([]byte(trimmed), &probe); err != nil {
		return nil
	}
	return probe.BackfillTimeList
}

// NormalizeDate reduces a DS date/datetime string ("2026-09-20 03:00:00" or
// "2026-09-20") to its yyyy-MM-dd part. It returns "" when the input has no
// recognisable date prefix.
func NormalizeDate(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) < 10 {
		return ""
	}
	candidate := s[:10]
	if _, err := time.Parse("2006-01-02", candidate); err != nil {
		return ""
	}
	return candidate
}

// DateRange returns the inclusive list of yyyy-MM-dd dates between start and
// end. Both bounds accept a date or datetime string.
func DateRange(start, end string) ([]string, error) {
	s := NormalizeDate(start)
	e := NormalizeDate(end)
	if s == "" || e == "" {
		return nil, fmt.Errorf("invalid date range %q..%q: expect yyyy-MM-dd or yyyy-MM-dd HH:mm:ss", start, end)
	}
	from, _ := time.Parse("2006-01-02", s)
	to, _ := time.Parse("2006-01-02", e)
	if from.After(to) {
		return nil, fmt.Errorf("start date %s is after end date %s", s, e)
	}
	var out []string
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out, nil
}

// BackfillDateState is the per-date coverage detail. DS materialises one
// workflow instance per schedule date and, once an instance succeeds, spawns
// the next instance with the remaining dates; so the date an instance actually
// ran is its own scheduleTime, while backfillTimeList is only the intent
// carried by that request.
type BackfillDateState struct {
	Date          string   `json:"date"`
	ExecutedBy    []int    `json:"executed_by"`
	ExecutedState []string `json:"executed_states"`
	ReferencedBy  []int    `json:"referenced_by,omitempty"`
}

// BackfillCoverage summarises how well a set of backfill instances covers an
// expected date range.
type BackfillCoverage struct {
	ExpectedDays     int                 `json:"expected_days"`
	ExecutedDays     int                 `json:"executed_days"`
	SuccessDays      int                 `json:"success_days"`
	MissingDates     []string            `json:"missing_dates"`
	NotExecutedDates []string            `json:"not_executed_dates"`
	NotSuccessDates  []string            `json:"not_success_dates"`
	DuplicateDates   []string            `json:"duplicate_dates"`
	ExtraDates       []string            `json:"extra_dates"`
	StateCounts      map[string]int      `json:"state_counts"`
	Instances        int                 `json:"instances"`
	Dates            []BackfillDateState `json:"dates,omitempty"`
}

// ComputeBackfillCoverage compares the expected dates against what the given
// instances actually ran. expected must contain yyyy-MM-dd strings.
//
// missing_dates     - never referenced by any instance (never requested)
// not_executed_dates- referenced by a request but no instance ever ran that day
//
//	(the classic "chunk head blocked, successors never spawned"
//	failure mode)
//
// not_success_dates - ran, but no instance reached SUCCESS
// duplicate_dates   - ran more than once
func ComputeBackfillCoverage(expected []string, instances []WorkflowInstanceSummary) BackfillCoverage {
	cov := BackfillCoverage{
		StateCounts:      map[string]int{},
		MissingDates:     []string{},
		NotExecutedDates: []string{},
		NotSuccessDates:  []string{},
		DuplicateDates:   []string{},
		ExtraDates:       []string{},
		Dates:            []BackfillDateState{},
		Instances:        len(instances),
	}
	perDate := map[string]*BackfillDateState{}
	executedStates := map[string][]string{}
	touch := func(d string) *BackfillDateState {
		if st, ok := perDate[d]; ok {
			return st
		}
		st := &BackfillDateState{Date: d}
		perDate[d] = st
		return st
	}
	for _, inst := range instances {
		cov.StateCounts[inst.State]++
		if own := NormalizeDate(inst.ScheduleTime); own != "" {
			st := touch(own)
			st.ExecutedBy = append(st.ExecutedBy, inst.ID)
			st.ExecutedState = append(st.ExecutedState, inst.State)
			executedStates[own] = append(executedStates[own], inst.State)
		}
		for _, d := range inst.ReferencedDates() {
			st := touch(d)
			st.ReferencedBy = append(st.ReferencedBy, inst.ID)
		}
	}
	expectedSet := map[string]bool{}
	for _, d := range expected {
		expectedSet[d] = true
	}
	cov.ExpectedDays = len(expected)
	for _, d := range expected {
		st, ok := perDate[d]
		if !ok || len(st.ReferencedBy) == 0 && len(st.ExecutedBy) == 0 {
			cov.MissingDates = append(cov.MissingDates, d)
			continue
		}
		if len(st.ExecutedBy) == 0 {
			cov.NotExecutedDates = append(cov.NotExecutedDates, d)
			continue
		}
		cov.ExecutedDays++
		if hasSuccess(executedStates[d]) {
			cov.SuccessDays++
		} else {
			cov.NotSuccessDates = append(cov.NotSuccessDates, d)
		}
		if len(st.ExecutedBy) > 1 {
			cov.DuplicateDates = append(cov.DuplicateDates, d)
		}
	}
	for d := range perDate {
		if !expectedSet[d] {
			cov.ExtraDates = append(cov.ExtraDates, d)
		}
	}
	sort.Strings(cov.MissingDates)
	sort.Strings(cov.NotExecutedDates)
	sort.Strings(cov.NotSuccessDates)
	sort.Strings(cov.DuplicateDates)
	sort.Strings(cov.ExtraDates)
	for _, d := range expected {
		if st, ok := perDate[d]; ok {
			cov.Dates = append(cov.Dates, *st)
		}
	}
	for _, d := range cov.ExtraDates {
		cov.Dates = append(cov.Dates, *perDate[d])
	}
	return cov
}

func hasSuccess(states []string) bool {
	for _, s := range states {
		if s == "SUCCESS" {
			return true
		}
	}
	return false
}

// ListWorkflowInstances fetches a single page of workflow instances.
func ListWorkflowInstances(ctx context.Context, client *Client, projectCode, workflowCode int64, pageNo, pageSize int) ([]WorkflowInstanceSummary, int, error) {
	return ListWorkflowInstancesWithOptions(ctx, client, projectCode, WorkflowInstanceListOptions{
		WorkflowCode: workflowCode,
	}, pageNo, pageSize)
}

// ListWorkflowInstancesWithOptions fetches a page while preserving the
// endpoint's server-side filters. This is used by compact/all CLI modes too.
func ListWorkflowInstancesWithOptions(ctx context.Context, client *Client, projectCode int64, options WorkflowInstanceListOptions, pageNo, pageSize int) ([]WorkflowInstanceSummary, int, error) {
	values := url.Values{}
	if options.WorkflowCode != 0 {
		values.Set("workflowDefinitionCode", strconv.FormatInt(options.WorkflowCode, 10))
	}
	values.Set("searchVal", options.Search)
	values.Set("executorName", options.ExecutorName)
	values.Set("stateType", options.StateType)
	values.Set("host", options.Host)
	values.Set("startDate", options.StartDate)
	values.Set("endDate", options.EndDate)
	values.Set("pageNo", strconv.Itoa(pageNo))
	values.Set("pageSize", strconv.Itoa(pageSize))
	resp, err := client.Form(ctx, http.MethodGet, fmt.Sprintf("/projects/%d/workflow-instances", projectCode), values)
	if err != nil {
		return nil, 0, err
	}
	return decodeWorkflowInstancePage(resp.Body)
}

func decodeWorkflowInstancePage(body []byte) ([]WorkflowInstanceSummary, int, error) {
	var decoded struct {
		Data struct {
			Total     int                       `json:"total"`
			TotalList []WorkflowInstanceSummary `json:"totalList"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, 0, fmt.Errorf("decode workflow instance page: %w", err)
	}
	return decoded.Data.TotalList, decoded.Data.Total, nil
}

// ListAllWorkflowInstances pages through every workflow instance in a project,
// optionally filtered by workflow definition code.
func ListAllWorkflowInstances(ctx context.Context, client *Client, projectCode, workflowCode int64, pageSize int) ([]WorkflowInstanceSummary, error) {
	return ListAllWorkflowInstancesWithOptions(ctx, client, projectCode, WorkflowInstanceListOptions{
		WorkflowCode: workflowCode,
	}, pageSize)
}

// ListAllWorkflowInstancesWithOptions pages through all filtered instances.
func ListAllWorkflowInstancesWithOptions(ctx context.Context, client *Client, projectCode int64, options WorkflowInstanceListOptions, pageSize int) ([]WorkflowInstanceSummary, error) {
	if pageSize <= 0 {
		pageSize = 200
	}
	var all []WorkflowInstanceSummary
	for page := 1; ; page++ {
		rows, total, err := ListWorkflowInstancesWithOptions(ctx, client, projectCode, options, page, pageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, rows...)
		if len(rows) == 0 || len(all) >= total {
			return all, nil
		}
	}
}
