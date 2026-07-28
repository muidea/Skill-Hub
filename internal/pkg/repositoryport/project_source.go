package repositoryport

import (
	"fmt"

	repositorycommon "github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/common"
	repositoryevents "github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/events"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/skill-hub/pkg/spec"
)

// ProjectSource is the repository data required by project-side use cases.
// It intentionally exposes repository values rather than a Repository service.
type ProjectSource interface {
	Default() (string, error)
	Path(name string) (string, error)
	Archive(skillID, sourcePath string) error
	FindSkill(skillID string) ([]spec.SkillMetadata, error)
	LoadSkill(skillID, repository string) (*spec.Skill, error)
	RebuildIndex(repository string) error
}

type eventHubProjectSource struct {
	hub    event.Hub
	source string
}

var _ ProjectSource = (*eventHubProjectSource)(nil)

// NewProjectSource creates an EventHub-backed project repository port.
func NewProjectSource(hub event.Hub, source string) ProjectSource {
	return &eventHubProjectSource{hub: hub, source: source}
}

func (p *eventHubProjectSource) Default() (string, error) {
	result := p.hub.Send(event.NewEvent(
		repositoryevents.TopicDefaultRepository, p.source, repositorycommon.ModuleName, nil,
		repositoryevents.DefaultRepositoryCommand{},
	))
	value, err := event.GetAs[repositoryevents.DefaultRepositoryResult](result)
	if err != nil {
		return "", err
	}
	if value.Repository.Name == "" {
		return "", fmt.Errorf("default repository is unavailable")
	}
	return value.Repository.Name, nil
}

func (p *eventHubProjectSource) Path(name string) (string, error) {
	result := p.hub.Send(event.NewEvent(
		repositoryevents.TopicRepositoryPath, p.source, repositorycommon.ModuleName, nil,
		repositoryevents.RepositoryPathCommand{Name: name},
	))
	value, err := event.GetAs[repositoryevents.RepositoryPathResult](result)
	if err != nil {
		return "", err
	}
	return value.Path, nil
}

func (p *eventHubProjectSource) Archive(skillID, sourcePath string) error {
	result := p.hub.Send(event.NewEvent(
		repositoryevents.TopicArchiveSkill, p.source, repositorycommon.ModuleName, nil,
		repositoryevents.ArchiveSkillCommand{SkillID: skillID, SourcePath: sourcePath},
	))
	if result == nil {
		return fmt.Errorf("repository archive returned no result")
	}
	_, err := event.GetAs[repositoryevents.ArchiveSkillResult](result)
	if err != nil {
		return err
	}
	return nil
}

func (p *eventHubProjectSource) FindSkill(skillID string) ([]spec.SkillMetadata, error) {
	result := p.hub.Send(event.NewEvent(
		repositoryevents.TopicFindSkill, p.source, repositorycommon.ModuleName, nil,
		repositoryevents.FindSkillCommand{SkillID: skillID},
	))
	value, err := event.GetAs[repositoryevents.FindSkillResult](result)
	if err != nil {
		return nil, err
	}
	return value.Skills, nil
}

func (p *eventHubProjectSource) LoadSkill(skillID, repository string) (*spec.Skill, error) {
	result := p.hub.Send(event.NewEvent(
		repositoryevents.TopicLoadSkill, p.source, repositorycommon.ModuleName, nil,
		repositoryevents.LoadSkillCommand{SkillID: skillID, RepoName: repository},
	))
	value, err := event.GetAs[repositoryevents.LoadSkillResult](result)
	if err != nil {
		return nil, err
	}
	return &value.Skill, nil
}

func (p *eventHubProjectSource) RebuildIndex(repository string) error {
	result := p.hub.Send(event.NewEvent(
		repositoryevents.TopicRebuildIndex, p.source, repositorycommon.ModuleName, nil,
		repositoryevents.RebuildIndexCommand{Repository: repository},
	))
	if result == nil {
		return fmt.Errorf("repository rebuild-index returned no result")
	}
	_, err := event.GetAs[repositoryevents.RebuildIndexResult](result)
	if err != nil {
		return err
	}
	return nil
}
