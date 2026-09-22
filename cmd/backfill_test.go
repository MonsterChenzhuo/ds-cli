package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ds-cli/ds-cli/internal/dsapi"
)

func TestSplitDateListKeepsDatetimeEntries(t *testing.T) {
	got := splitDateList("2025-01-01 03:00:00, 2025-01-02\n2025-01-03 ;2025-01-04")
	want := []string{"2025-01-01 03:00:00", "2025-01-02", "2025-01-03", "2025-01-04"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBuildComplementScheduleTimeDateList(t *testing.T) {
	got, err := buildComplementScheduleTime("2025-01-02,2025-01-03 04:05:06", "", "", "", "03:00:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload struct {
		List string `json:"complementScheduleDateList"`
	}
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("invalid JSON %q: %v", got, err)
	}
	want := "2025-01-02 03:00:00,2025-01-03 04:05:06"
	if payload.List != want {
		t.Fatalf("complementScheduleDateList = %q, want %q", payload.List, want)
	}
}

func TestBuildComplementScheduleTimeFromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dates.txt")
	if err := os.WriteFile(file, []byte("2025-01-02\n2025-01-03\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := buildComplementScheduleTime("", file, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload struct {
		List string `json:"complementScheduleDateList"`
	}
	_ = json.Unmarshal([]byte(got), &payload)
	if payload.List != "2025-01-02 00:00:00,2025-01-03 00:00:00" {
		t.Fatalf("list = %q", payload.List)
	}
}

func TestBuildComplementScheduleTimeRange(t *testing.T) {
	got, err := buildComplementScheduleTime("", "", "2025-01-01", "2026-09-20", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload struct {
		Start string `json:"complementStartDate"`
		End   string `json:"complementEndDate"`
	}
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("invalid JSON %q: %v", got, err)
	}
	if payload.Start != "2025-01-01 00:00:00" || payload.End != "2026-09-20 00:00:00" {
		t.Fatalf("range = %q..%q", payload.Start, payload.End)
	}
}

func TestBuildComplementScheduleTimeRejectsBadInput(t *testing.T) {
	cases := []struct {
		name                   string
		list, file, start, end string
	}{
		{"nothing", "", "", "", ""},
		{"list-and-file", "2025-01-01", "f", "", ""},
		{"list-and-range", "2025-01-01", "", "2025-01-01", "2025-01-02"},
		{"range-half", "", "", "2025-01-01", ""},
		{"reversed-range", "", "", "2026-01-01", "2025-01-01"},
		{"bad-date", "not-a-date", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := tc.file
			if file == "f" {
				file = filepath.Join(t.TempDir(), "f")
				_ = os.WriteFile(file, []byte("2025-01-01\n"), 0o600)
			}
			if _, err := buildComplementScheduleTime(tc.list, file, tc.start, tc.end, ""); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestResolveExpectedDates(t *testing.T) {
	got, err := resolveExpectedDates("2025-01-01", "2025-01-03", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"2025-01-01", "2025-01-02", "2025-01-03"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	got, err = resolveExpectedDates("", "", "2025-01-01 03:00:00,2025-02-01", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"2025-01-01", "2025-02-01"}) {
		t.Fatalf("got %v", got)
	}
	if _, err := resolveExpectedDates("", "", "", ""); err == nil {
		t.Fatal("expected error when nothing provided")
	}
	if _, err := resolveExpectedDates("2025-01-01", "2025-01-02", "2025-01-03", ""); err == nil {
		t.Fatal("expected error for range+list")
	}
}

func TestFilterWorkflowInstancesByCmdType(t *testing.T) {
	rows := []dsapi.WorkflowInstanceSummary{
		{ID: 1, CmdTypeIfComplement: "COMPLEMENT_DATA"},
		{ID: 2, CmdTypeIfComplement: "SCHEDULER"},
		{ID: 3, CommandType: "START_PROCESS"},
	}
	got := filterWorkflowInstancesByCmdType(rows, "complement_data")
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("got %+v", got)
	}
	got = filterWorkflowInstancesByCmdType(rows, "START_PROCESS")
	if len(got) != 1 || got[0].ID != 3 {
		t.Fatalf("got %+v", got)
	}
}
