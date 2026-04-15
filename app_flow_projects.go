// 本文件负责持久化 Flow 项目草稿，并校验 Flow 页面使用的项目目录。

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	flowProjectsStateKey         = "flow.projects.state.v1"
	flowProjectsStateVersion     = 1
	flowProjectIDMaxLen          = 128
	flowProjectFlowIDMaxLen      = 128
	flowProjectNameMaxLen        = 128
	flowProjectDescriptionMaxLen = 1024
	flowProjectUpdatedAtMaxLen   = 64
)

type FlowProjectsState struct {
	Version          int           `json:"version"`
	CurrentProjectID string        `json:"current_project_id,omitempty"`
	Projects         []FlowProject `json:"projects"`
}

type FlowProject struct {
	ProjectID   string         `json:"project_id"`
	FlowID      string         `json:"flow_id"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	UpdatedAt   string         `json:"updated_at,omitempty"`
	Trigger     map[string]any `json:"trigger,omitempty"`
	Graph       map[string]any `json:"graph,omitempty"`
}

func (a *App) FlowProjectsState() (FlowProjectsState, error) {
	if a.store == nil {
		return FlowProjectsState{}, errors.New("storage not initialized")
	}
	profile := a.store.CurrentProfile()
	raw := a.store.GetString(profile, flowProjectsStateKey, "")
	state := normalizeFlowProjectsState(parseFlowProjectsState(raw))
	if err := validateFlowProjectsState(state); err != nil {
		return FlowProjectsState{}, err
	}
	return state, nil
}

func (a *App) SaveFlowProjectsState(state FlowProjectsState) (FlowProjectsState, error) {
	if a.store == nil {
		return FlowProjectsState{}, errors.New("storage not initialized")
	}
	normalized := normalizeFlowProjectsState(state)
	if err := validateFlowProjectsState(normalized); err != nil {
		return FlowProjectsState{}, err
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return FlowProjectsState{}, err
	}
	profile := a.store.CurrentProfile()
	if err := a.store.SetString(profile, flowProjectsStateKey, string(data)); err != nil {
		return FlowProjectsState{}, err
	}
	return a.FlowProjectsState()
}

func parseFlowProjectsState(raw string) FlowProjectsState {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return FlowProjectsState{}
	}
	var state FlowProjectsState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return FlowProjectsState{}
	}
	return state
}

func normalizeFlowProjectsState(state FlowProjectsState) FlowProjectsState {
	if state.Version <= 0 {
		state.Version = flowProjectsStateVersion
	}
	state.CurrentProjectID = strings.TrimSpace(state.CurrentProjectID)

	projects := make([]FlowProject, 0, len(state.Projects))
	seen := make(map[string]int, len(state.Projects))
	for _, project := range state.Projects {
		project = normalizeFlowProject(project)
		if project.ProjectID == "" || project.FlowID == "" {
			continue
		}
		if idx, ok := seen[project.ProjectID]; ok {
			projects[idx] = project
			continue
		}
		seen[project.ProjectID] = len(projects)
		projects = append(projects, project)
	}
	state.Projects = projects

	if state.CurrentProjectID != "" && !containsFlowProjectID(state.Projects, state.CurrentProjectID) {
		state.CurrentProjectID = ""
	}
	if state.CurrentProjectID == "" && len(state.Projects) > 0 {
		state.CurrentProjectID = state.Projects[0].ProjectID
	}

	return state
}

func normalizeFlowProject(project FlowProject) FlowProject {
	project.ProjectID = strings.TrimSpace(project.ProjectID)
	project.FlowID = strings.TrimSpace(project.FlowID)
	project.Name = strings.TrimSpace(project.Name)
	project.Description = strings.TrimSpace(project.Description)
	project.UpdatedAt = strings.TrimSpace(project.UpdatedAt)
	return project
}

func validateFlowProjectsState(state FlowProjectsState) error {
	for idx, project := range state.Projects {
		if err := validateFlowProject(project); err != nil {
			return fmt.Errorf("projects[%d]: %w", idx, err)
		}
	}
	if state.CurrentProjectID != "" && !containsFlowProjectID(state.Projects, state.CurrentProjectID) {
		return errors.New("current_project_id not found in projects")
	}
	return nil
}

func validateFlowProject(project FlowProject) error {
	if project.ProjectID == "" {
		return errors.New("project_id is required")
	}
	if project.FlowID == "" {
		return errors.New("flow_id is required")
	}
	if len(project.ProjectID) > flowProjectIDMaxLen {
		return errors.New("project_id is too long")
	}
	if len(project.FlowID) > flowProjectFlowIDMaxLen {
		return errors.New("flow_id is too long")
	}
	if len(project.Name) > flowProjectNameMaxLen {
		return errors.New("name is too long")
	}
	if len(project.Description) > flowProjectDescriptionMaxLen {
		return errors.New("description is too long")
	}
	if len(project.UpdatedAt) > flowProjectUpdatedAtMaxLen {
		return errors.New("updated_at is too long")
	}
	return nil
}

func containsFlowProjectID(projects []FlowProject, projectID string) bool {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return false
	}
	for _, project := range projects {
		if project.ProjectID == projectID {
			return true
		}
	}
	return false
}
