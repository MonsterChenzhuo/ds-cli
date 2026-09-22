package dsapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNormalizeDate(t *testing.T) {
	cases := map[string]string{
		"2026-09-20 03:00:00":  "2026-09-20",
		"2026-09-20":           "2026-09-20",
		" 2025-01-01 03:00:00": "2025-01-01",
		"":                     "",
		"not-a-date":           "",
		"2026-13-01":           "",
	}
	for in, want := range cases {
		if got := NormalizeDate(in); got != want {
			t.Errorf("NormalizeDate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDateRangeInclusive(t *testing.T) {
	got, err := DateRange("2025-01-01 03:00:00", "2025-01-04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"2025-01-01", "2025-01-02", "2025-01-03", "2025-01-04"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if _, err := DateRange("2025-01-05", "2025-01-01"); err == nil {
		t.Fatal("expected error for reversed range")
	}
	if _, err := DateRange("bad", "2025-01-01"); err == nil {
		t.Fatal("expected error for invalid bound")
	}
}

func TestReferencedDatesFromBackfillTimeList(t *testing.T) {
	inst := WorkflowInstanceSummary{
		ScheduleTime: "2025-01-05 03:00:00",
		CommandParam: `{"commandType":"COMPLEMENT_DATA","backfillTimeList":["2025-01-06 03:00:00","2025-01-05 03:00:00","2025-01-06 03:00:00"]}`,
	}
	got := inst.ReferencedDates()
	want := []string{"2025-01-05", "2025-01-06"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReferencedDatesIgnoresScheduleTimeAndBadJSON(t *testing.T) {
	// scheduleTime is what the instance ran, not what it referenced; a manual or
	// scheduled run carries no backfillTimeList.
	manual := WorkflowInstanceSummary{ScheduleTime: "2026-09-20 03:00:00", CommandParam: `{"commandType":"START_PROCESS"}`}
	if got := manual.ReferencedDates(); len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
	if got := (WorkflowInstanceSummary{CommandParam: "not-json"}).ReferencedDates(); len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
	if got := (WorkflowInstanceSummary{}).ReferencedDates(); len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestIsComplement(t *testing.T) {
	if !(WorkflowInstanceSummary{CmdTypeIfComplement: "COMPLEMENT_DATA"}).IsComplement() {
		t.Fatal("cmdTypeIfComplement should mark complement")
	}
	if !(WorkflowInstanceSummary{CommandType: "COMPLEMENT_DATA"}).IsComplement() {
		t.Fatal("commandType should mark complement")
	}
	if (WorkflowInstanceSummary{CmdTypeIfComplement: "SCHEDULER"}).IsComplement() {
		t.Fatal("scheduler instance must not be complement")
	}
}

func TestComputeBackfillCoverage(t *testing.T) {
	mk := func(id int, state, schedule string, refs ...string) WorkflowInstanceSummary {
		cp := map[string]any{}
		if len(refs) > 0 {
			cp["backfillTimeList"] = refs
		}
		raw, _ := json.Marshal(cp)
		return WorkflowInstanceSummary{ID: id, State: state, ScheduleTime: schedule, CommandParam: string(raw)}
	}
	expected := []string{"2025-01-01", "2025-01-02", "2025-01-03", "2025-01-04", "2025-01-05"}
	instances := []WorkflowInstanceSummary{
		mk(1, "SUCCESS", "2025-01-01 03:00:00"),
		mk(2, "SUCCESS", "2025-01-02 03:00:00"),
		mk(3, "RUNNING_EXECUTION", "2025-01-03 03:00:00"),
		// id4 ran 2025-02-01 but its request also referenced 01-02 and 01-04
		mk(4, "SUCCESS", "2025-02-01 03:00:00", "2025-01-02 03:00:00", "2025-01-04 03:00:00", "2025-02-01 03:00:00"),
	}
	cov := ComputeBackfillCoverage(expected, instances)
	if cov.ExpectedDays != 5 || cov.ExecutedDays != 3 || cov.SuccessDays != 2 {
		t.Fatalf("unexpected counts: %+v", cov)
	}
	if !reflect.DeepEqual(cov.MissingDates, []string{"2025-01-05"}) {
		t.Fatalf("missing = %v", cov.MissingDates)
	}
	if !reflect.DeepEqual(cov.NotExecutedDates, []string{"2025-01-04"}) {
		t.Fatalf("not-executed = %v", cov.NotExecutedDates)
	}
	if !reflect.DeepEqual(cov.NotSuccessDates, []string{"2025-01-03"}) {
		t.Fatalf("not-success = %v", cov.NotSuccessDates)
	}
	if !reflect.DeepEqual(cov.DuplicateDates, []string{}) {
		t.Fatalf("duplicate = %v", cov.DuplicateDates)
	}
	if !reflect.DeepEqual(cov.ExtraDates, []string{"2025-02-01"}) {
		t.Fatalf("extra = %v", cov.ExtraDates)
	}
	if cov.StateCounts["SUCCESS"] != 3 || cov.StateCounts["RUNNING_EXECUTION"] != 1 {
		t.Fatalf("state counts = %v", cov.StateCounts)
	}
	if len(cov.Dates) != 5 {
		t.Fatalf("dates detail len = %d, want 5 expected dates", len(cov.Dates))
	}
}

func TestComputeBackfillCoverageChainReferenceIsNotExecution(t *testing.T) {
	// A blocked chunk head: the head instance references 01-01..01-03 but only
	// 01-01 ever ran, so 01-02/01-03 must be reported as not executed (this is
	// exactly the failure mode that silently dropped 61 days in production).
	head := WorkflowInstanceSummary{
		ID:           1,
		State:        "RUNNING_EXECUTION",
		ScheduleTime: "2025-01-01 03:00:00",
		CommandParam: `{"backfillTimeList":["2025-01-01 03:00:00","2025-01-02 03:00:00","2025-01-03 03:00:00"]}`,
	}
	cov := ComputeBackfillCoverage(
		[]string{"2025-01-01", "2025-01-02", "2025-01-03"},
		[]WorkflowInstanceSummary{head},
	)
	if len(cov.MissingDates) != 0 {
		t.Fatalf("missing = %v, want none", cov.MissingDates)
	}
	if !reflect.DeepEqual(cov.NotExecutedDates, []string{"2025-01-02", "2025-01-03"}) {
		t.Fatalf("not-executed = %v", cov.NotExecutedDates)
	}
}
