// Package projectstateport defines the typed project-state capability used by
// application use cases. Production implementations communicate with the
// project-state Block through its EventHub contracts.
package projectstateport

import (
	"fmt"

	projectstatecommon "github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/common"
	projectstateevents "github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/events"
	"github.com/muidea/skill-hub/pkg/spec"

	"github.com/muidea/magicCommon/event"
)

// ProjectState provides the project-state operations needed outside the
// project-state Block. It deliberately returns values, never a StateManager.
type ProjectState interface {
	Find(projectPath string) (*spec.ProjectState, error)
	Load(projectPath string) (*spec.ProjectState, error)
	List() (map[string]spec.ProjectState, error)
	Save(project spec.ProjectState) error
	HasSkill(projectPath, skillID string) (bool, error)
	RemoveSkill(projectPath, skillID string) error
	PruneInvalidProjects() ([]string, error)
}

type eventHubPort struct {
	hub    event.Hub
	source string
}

var _ ProjectState = (*eventHubPort)(nil)

// New creates an EventHub-backed project-state port.
func New(hub event.Hub, source string) ProjectState {
	return &eventHubPort{hub: hub, source: source}
}

func (p *eventHubPort) Find(projectPath string) (*spec.ProjectState, error) {
	result := p.hub.Send(event.NewEvent(
		projectstateevents.TopicLoadProject, p.source, projectstatecommon.ModuleName, nil,
		projectstateevents.LoadProjectCommand{ProjectPath: projectPath},
	))
	value, err := event.GetAs[projectstateevents.LoadProjectResult](result)
	if err != nil {
		return nil, err
	}
	if !value.Found {
		return nil, nil
	}
	return &value.Project, nil
}

func (p *eventHubPort) Load(projectPath string) (*spec.ProjectState, error) {
	result := p.hub.Send(event.NewEvent(
		projectstateevents.TopicLoadProjectState, p.source, projectstatecommon.ModuleName, nil,
		projectstateevents.LoadProjectStateCommand{ProjectPath: projectPath},
	))
	value, err := event.GetAs[projectstateevents.LoadProjectStateResult](result)
	if err != nil {
		return nil, err
	}
	return &value.Project, nil
}

func (p *eventHubPort) Save(project spec.ProjectState) error {
	result := p.hub.Send(event.NewEvent(
		projectstateevents.TopicSaveProject, p.source, projectstatecommon.ModuleName, nil,
		projectstateevents.SaveProjectCommand{Project: project},
	))
	if result == nil {
		return fmt.Errorf("project-state save returned no result")
	}
	_, err := event.GetAs[projectstateevents.SaveProjectResult](result)
	if err != nil {
		return err
	}
	return nil
}

func (p *eventHubPort) List() (map[string]spec.ProjectState, error) {
	result := p.hub.Send(event.NewEvent(
		projectstateevents.TopicListProjects, p.source, projectstatecommon.ModuleName, nil,
		projectstateevents.ListProjectsCommand{},
	))
	value, err := event.GetAs[projectstateevents.ListProjectsResult](result)
	if err != nil {
		return nil, err
	}
	return value.Projects, nil
}

func (p *eventHubPort) HasSkill(projectPath, skillID string) (bool, error) {
	result := p.hub.Send(event.NewEvent(
		projectstateevents.TopicProjectHasSkill, p.source, projectstatecommon.ModuleName, nil,
		projectstateevents.ProjectHasSkillCommand{ProjectPath: projectPath, SkillID: skillID},
	))
	value, err := event.GetAs[projectstateevents.ProjectHasSkillResult](result)
	if err != nil {
		return false, err
	}
	return value.HasSkill, nil
}

func (p *eventHubPort) RemoveSkill(projectPath, skillID string) error {
	result := p.hub.Send(event.NewEvent(
		projectstateevents.TopicRemoveSkill, p.source, projectstatecommon.ModuleName, nil,
		projectstateevents.RemoveSkillCommand{ProjectPath: projectPath, SkillID: skillID},
	))
	_, err := event.GetAs[projectstateevents.RemoveSkillResult](result)
	if err != nil {
		return err
	}
	return nil
}

func (p *eventHubPort) PruneInvalidProjects() ([]string, error) {
	result := p.hub.Send(event.NewEvent(
		projectstateevents.TopicPruneProjects, p.source, projectstatecommon.ModuleName, nil,
		projectstateevents.PruneProjectsCommand{},
	))
	value, err := event.GetAs[projectstateevents.PruneProjectsResult](result)
	if err != nil {
		return nil, err
	}
	return value.Removed, nil
}
