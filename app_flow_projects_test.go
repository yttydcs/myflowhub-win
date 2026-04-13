// Context: covers the flow projects app binding helpers and persistence rules.

package main

import "testing"

func TestParseFlowProjectsState_Empty(t *testing.T) {
	if out := parseFlowProjectsState(""); len(out.Projects) != 0 || out.Version != 0 {
		t.Fatalf("expected zero state got %+v", out)
	}
}

func TestParseFlowProjectsState_Invalid(t *testing.T) {
	if out := parseFlowProjectsState("{"); len(out.Projects) != 0 || out.Version != 0 {
		t.Fatalf("expected zero state got %+v", out)
	}
}

func TestNormalizeFlowProjectsState_TrimsFiltersAndDedupes(t *testing.T) {
	out := normalizeFlowProjectsState(FlowProjectsState{
		CurrentProjectID: " p2 ",
		Projects: []FlowProject{
			{ProjectID: " p1 ", FlowID: " f1 ", Name: " Draft "},
			{ProjectID: "", FlowID: "f-ignore"},
			{ProjectID: "p2", FlowID: ""},
			{ProjectID: "p1", FlowID: "f1-updated", Name: " Updated "},
		},
	})

	if out.Version != flowProjectsStateVersion {
		t.Fatalf("expected version=%d got %d", flowProjectsStateVersion, out.Version)
	}
	if len(out.Projects) != 1 {
		t.Fatalf("expected 1 project got %d", len(out.Projects))
	}
	project := out.Projects[0]
	if project.ProjectID != "p1" || project.FlowID != "f1-updated" || project.Name != "Updated" {
		t.Fatalf("unexpected normalized project: %+v", project)
	}
	if out.CurrentProjectID != "p1" {
		t.Fatalf("expected current_project_id fallback to p1 got %q", out.CurrentProjectID)
	}
}

func TestValidateFlowProjectsState_ProjectIDTooLong(t *testing.T) {
	tooLong := ""
	for i := 0; i < flowProjectIDMaxLen+1; i++ {
		tooLong += "a"
	}
	err := validateFlowProjectsState(FlowProjectsState{
		Projects: []FlowProject{
			{ProjectID: tooLong, FlowID: "f1"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
