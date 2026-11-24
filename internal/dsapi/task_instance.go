package dsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type TaskInstanceSummary struct {
	ID                   int    `json:"id"`
	Name                 string `json:"name"`
	State                string `json:"state"`
	TaskType             string `json:"taskType"`
	TaskExecuteType      string `json:"taskExecuteType"`
	WorkflowInstanceID   int    `json:"workflowInstanceId"`
	WorkflowInstanceName string `json:"workflowInstanceName,omitempty"`
	TaskCode             int64  `json:"taskCode,omitempty"`
	StartTime            string `json:"startTime,omitempty"`
	EndTime              string `json:"endTime,omitempty"`
}

type TaskInstancePage struct {
	Total     int               `json:"total"`
	TotalPage int               `json:"totalPage"`
	TotalList []json.RawMessage `json:"totalList"`
}

func QueryTaskInstances(ctx context.Context, client *Client, projectCode int64, filters url.Values, pageNo, pageSize int) (*Response, error) {
	query := url.Values{
		"pageNo":   {strconv.Itoa(pageNo)},
		"pageSize": {strconv.Itoa(pageSize)},
	}
	for key, values := range filters {
		query[key] = values
	}
	return client.Form(ctx, http.MethodGet, fmt.Sprintf("/projects/%d/task-instances", projectCode), query)
}

func DecodeTaskInstancePage(body []byte) (TaskInstancePage, error) {
	var result struct {
		Data struct {
			Total     *int              `json:"total"`
			TotalPage int               `json:"totalPage"`
			TotalList []json.RawMessage `json:"totalList"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return TaskInstancePage{}, fmt.Errorf("decode task instance page: %w", err)
	}
	if result.Data.Total == nil || *result.Data.Total < 0 || result.Data.TotalPage < 0 {
		return TaskInstancePage{}, fmt.Errorf("task instance list response has invalid pagination data")
	}
	return TaskInstancePage{Total: *result.Data.Total, TotalPage: result.Data.TotalPage, TotalList: result.Data.TotalList}, nil
}

func ListAllTaskInstances(ctx context.Context, client *Client, projectCode int64, filters url.Values, pageSize int) (TaskInstancePage, error) {
	all := TaskInstancePage{TotalList: []json.RawMessage{}}
	seen := make(map[int]struct{})
	for pageNo := 1; ; pageNo++ {
		resp, err := QueryTaskInstances(ctx, client, projectCode, filters, pageNo, pageSize)
		if err != nil {
			return all, fmt.Errorf("list task instances page %d: %w", pageNo, err)
		}
		page, err := DecodeTaskInstancePage(resp.Body)
		if err != nil {
			return all, err
		}
		for _, raw := range page.TotalList {
			var row struct {
				ID int `json:"id"`
			}
			if err := json.Unmarshal(raw, &row); err != nil || row.ID <= 0 {
				return all, fmt.Errorf("task instance list page %d contains a row without a valid id", pageNo)
			}
			if _, exists := seen[row.ID]; !exists {
				seen[row.ID] = struct{}{}
				all.TotalList = append(all.TotalList, raw)
			}
		}
		all.Total, all.TotalPage = page.Total, page.TotalPage
		if (page.TotalPage > 0 && pageNo >= page.TotalPage) ||
			(page.TotalPage == 0 && (len(page.TotalList) == 0 || pageNo*pageSize >= page.Total)) {
			return all, nil
		}
	}
}

func SummarizeTaskInstances(rows []json.RawMessage) ([]TaskInstanceSummary, error) {
	summaries := make([]TaskInstanceSummary, 0, len(rows))
	for _, raw := range rows {
		var task TaskInstanceSummary
		if err := json.Unmarshal(raw, &task); err != nil {
			return nil, fmt.Errorf("decode task instance: %w", err)
		}
		summaries = append(summaries, task)
	}
	return summaries, nil
}

func StopStreamTaskInstance(ctx context.Context, client *Client, projectCode int64, taskInstanceID int) (*Response, error) {
	return client.Form(ctx, http.MethodPost,
		fmt.Sprintf("/projects/%d/task-instances/%d/stop", projectCode, taskInstanceID), nil)
}
