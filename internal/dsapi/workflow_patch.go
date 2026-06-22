package dsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// WorkflowPatchInput drives PatchWorkflowTask.
type WorkflowPatchInput struct {
	ProjectCode  int64
	WorkflowCode int64
	TaskCode     int64
	NewRawScript string
	// KeepOffline, when true, leaves the workflow OFFLINE after the update even if it was ONLINE before.
	KeepOffline bool
}

// WorkflowPatchResult summarizes what PatchWorkflowTask did so callers can include it in the CLI envelope.
type WorkflowPatchResult struct {
	PrevReleaseState   string          `json:"prev_release_state"`
	FinalReleaseState  string          `json:"final_release_state"`
	WorkflowName       string          `json:"workflow_name"`
	NewWorkflowVersion int             `json:"new_workflow_version"`
	UpdateResponse     json.RawMessage `json:"update_response"`
}

// PatchWorkflowTask fetches a workflow definition, swaps one task's rawScript, and updates the workflow
// definition via the legacy /projects/{}/workflow-definition/{code} PUT endpoint. If the workflow is
// ONLINE it is offlined before the update and (unless KeepOffline) onlined again afterwards.
func PatchWorkflowTask(ctx context.Context, client *Client, in WorkflowPatchInput) (*WorkflowPatchResult, error) {
	if in.ProjectCode == 0 || in.WorkflowCode == 0 || in.TaskCode == 0 {
		return nil, errors.New("project-code, workflow-code, and task-code are required")
	}
	if in.NewRawScript == "" {
		return nil, errors.New("new rawScript is required")
	}

	getResp, err := client.JSON(ctx, http.MethodGet,
		fmt.Sprintf("/projects/%d/workflow-definition/%d", in.ProjectCode, in.WorkflowCode), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch workflow definition: %w", err)
	}
	var full struct {
		Data struct {
			WorkflowDefinition       map[string]any   `json:"workflowDefinition"`
			WorkflowTaskRelationList []map[string]any `json:"workflowTaskRelationList"`
			TaskDefinitionList       []map[string]any `json:"taskDefinitionList"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getResp.Body, &full); err != nil {
		return nil, fmt.Errorf("decode workflow definition: %w", err)
	}
	wf := full.Data.WorkflowDefinition
	if wf == nil {
		return nil, errors.New("workflow definition payload missing 'workflowDefinition'")
	}

	tasks := full.Data.TaskDefinitionList
	if len(tasks) == 0 {
		return nil, errors.New("workflow has no task definitions to patch")
	}
	var found bool
	for _, t := range tasks {
		if asInt64(t["code"]) == in.TaskCode {
			params, _ := t["taskParams"].(map[string]any)
			if params == nil {
				params = map[string]any{}
			}
			params["rawScript"] = in.NewRawScript
			t["taskParams"] = params
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("task code %d not present in workflow %d", in.TaskCode, in.WorkflowCode)
	}

	cleanedTasks := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		cleanedTasks = append(cleanedTasks, stripDefinitionAuditFields(t,
			"id", "userName", "projectName", "createTime", "updateTime",
			"modifyBy", "operator", "operateTime", "taskParamList", "taskParamMap"))
	}
	cleanedRelations := make([]map[string]any, 0, len(full.Data.WorkflowTaskRelationList))
	for _, r := range full.Data.WorkflowTaskRelationList {
		cleanedRelations = append(cleanedRelations, stripDefinitionAuditFields(r,
			"id", "createTime", "updateTime", "operator", "operateTime"))
	}
	taskDefinitionJSON, err := json.Marshal(cleanedTasks)
	if err != nil {
		return nil, err
	}
	taskRelationJSON, err := json.Marshal(cleanedRelations)
	if err != nil {
		return nil, err
	}

	prevState, _ := wf["releaseState"].(string)
	wasOnline := prevState == "ONLINE"
	if wasOnline {
		if _, err := releaseWorkflowDefinition(ctx, client, in.ProjectCode, in.WorkflowCode, "OFFLINE"); err != nil {
			return nil, fmt.Errorf("offline workflow before update: %w", err)
		}
	}

	form := url.Values{}
	form.Set("name", stringField(wf, "name"))
	form.Set("description", stringField(wf, "description"))
	form.Set("globalParams", defaultString(stringField(wf, "globalParams"), "[]"))
	form.Set("locations", defaultString(stringField(wf, "locations"), "[]"))
	form.Set("timeout", strconv.FormatInt(asInt64(wf["timeout"]), 10))
	form.Set("executionType", defaultString(stringField(wf, "executionType"), "PARALLEL"))
	form.Set("taskDefinitionJson", string(taskDefinitionJSON))
	form.Set("taskRelationJson", string(taskRelationJSON))

	putResp, err := client.Form(ctx, http.MethodPut,
		fmt.Sprintf("/projects/%d/workflow-definition/%d", in.ProjectCode, in.WorkflowCode), form)
	if err != nil {
		if wasOnline {
			// Best-effort: restore ONLINE so we don't leave the workflow stuck OFFLINE on failure.
			_, _ = releaseWorkflowDefinition(ctx, client, in.ProjectCode, in.WorkflowCode, "ONLINE")
		}
		return nil, fmt.Errorf("update workflow definition: %w", err)
	}

	final := "OFFLINE"
	if wasOnline && !in.KeepOffline {
		if _, err := releaseWorkflowDefinition(ctx, client, in.ProjectCode, in.WorkflowCode, "ONLINE"); err != nil {
			return nil, fmt.Errorf("online workflow after update: %w", err)
		}
		final = "ONLINE"
	}

	res := &WorkflowPatchResult{
		PrevReleaseState:  prevState,
		FinalReleaseState: final,
		WorkflowName:      stringField(wf, "name"),
		UpdateResponse:    putResp.Body,
	}
	var putParsed struct {
		Data struct {
			Version int `json:"version"`
		} `json:"data"`
	}
	if json.Unmarshal(putResp.Body, &putParsed) == nil {
		res.NewWorkflowVersion = putParsed.Data.Version
	}
	return res, nil
}

// PatchTaskDefinition swaps a single task's rawScript via the with-upstream endpoint.
// upstreamCodes may be empty.
func PatchTaskDefinition(ctx context.Context, client *Client, projectCode, taskCode int64, newRawScript, upstreamCodes string) (*Response, error) {
	if projectCode == 0 || taskCode == 0 {
		return nil, errors.New("project-code and task-code are required")
	}
	if newRawScript == "" {
		return nil, errors.New("new rawScript is required")
	}
	getResp, err := client.JSON(ctx, http.MethodGet,
		fmt.Sprintf("/projects/%d/task-definition/%d", projectCode, taskCode), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch task definition: %w", err)
	}
	var full struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(getResp.Body, &full); err != nil {
		return nil, fmt.Errorf("decode task definition: %w", err)
	}
	if full.Data == nil {
		return nil, errors.New("task definition payload missing 'data'")
	}
	params, _ := full.Data["taskParams"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}
	params["rawScript"] = newRawScript
	full.Data["taskParams"] = params
	cleaned := stripDefinitionAuditFields(full.Data,
		"id", "userName", "projectName", "createTime", "updateTime",
		"modifyBy", "operator", "operateTime", "taskParamList", "taskParamMap")
	payload, err := json.Marshal(cleaned)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("taskDefinitionJsonObj", string(payload))
	if upstreamCodes != "" {
		form.Set("upstreamCodes", upstreamCodes)
	}
	return client.Form(ctx, http.MethodPut,
		fmt.Sprintf("/projects/%d/task-definition/%d/with-upstream", projectCode, taskCode), form)
}

func releaseWorkflowDefinition(ctx context.Context, client *Client, projectCode, workflowCode int64, state string) (*Response, error) {
	form := url.Values{}
	form.Set("releaseState", state)
	return client.Form(ctx, http.MethodPost,
		fmt.Sprintf("/projects/%d/workflow-definition/%d/release", projectCode, workflowCode), form)
}

func stripDefinitionAuditFields(in map[string]any, keys ...string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	for _, k := range keys {
		delete(out, k)
	}
	return out
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func defaultString(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func asInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}
