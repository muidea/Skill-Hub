package repositoryport

import (
	"fmt"

	"github.com/muidea/skill-hub/internal/config"
	gitpkg "github.com/muidea/skill-hub/internal/git"
	repositorycommon "github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/common"
	repositoryevents "github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/events"
	"github.com/muidea/skill-hub/pkg/spec"

	"github.com/muidea/magicCommon/event"
)

type eventHubRepository struct {
	hub    event.Hub
	source string
}

var _ Repository = (*eventHubRepository)(nil)

// NewRepository creates the production repository port for protocol adapters.
func NewRepository(hub event.Hub, source string) Repository {
	return &eventHubRepository{hub: hub, source: source}
}

func (p *eventHubRepository) Default() (*config.RepositoryConfig, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicDefaultRepository, p.source, repositorycommon.ModuleName, nil, repositoryevents.DefaultRepositoryCommand{}))
	v, err := event.GetAs[repositoryevents.DefaultRepositoryResult](r)
	if err != nil {
		return nil, err
	}
	return &v.Repository, nil
}
func (p *eventHubRepository) Get(name string) (*config.RepositoryConfig, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicGetRepository, p.source, repositorycommon.ModuleName, nil, repositoryevents.GetRepositoryCommand{Name: name}))
	v, err := event.GetAs[repositoryevents.GetRepositoryResult](r)
	if err != nil {
		return nil, err
	}
	return &v.Repository, nil
}
func (p *eventHubRepository) Path(name string) (string, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicRepositoryPath, p.source, repositorycommon.ModuleName, nil, repositoryevents.RepositoryPathCommand{Name: name}))
	v, err := event.GetAs[repositoryevents.RepositoryPathResult](r)
	if err != nil {
		return "", err
	}
	return v.Path, nil
}
func (p *eventHubRepository) List(includeDisabled bool) ([]config.RepositoryConfig, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicListRepositories, p.source, repositorycommon.ModuleName, nil, repositoryevents.ListRepositoriesCommand{IncludeDisabled: includeDisabled}))
	v, err := event.GetAs[repositoryevents.ListRepositoriesResult](r)
	if err != nil {
		return nil, err
	}
	return v.Repositories, nil
}
func (p *eventHubRepository) ListSkills(names []string) ([]spec.SkillMetadata, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicListSkills, p.source, repositorycommon.ModuleName, nil, repositoryevents.ListSkillsCommand{Repositories: names}))
	v, err := event.GetAs[repositoryevents.ListSkillsResult](r)
	if err != nil {
		return nil, err
	}
	return v.Skills, nil
}
func (p *eventHubRepository) FindSkill(id string) ([]spec.SkillMetadata, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicFindSkill, p.source, repositorycommon.ModuleName, nil, repositoryevents.FindSkillCommand{SkillID: id}))
	v, err := event.GetAs[repositoryevents.FindSkillResult](r)
	if err != nil {
		return nil, err
	}
	return v.Skills, nil
}
func (p *eventHubRepository) FindSkills(patterns, names []string) ([]spec.SkillMetadata, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicFindSkills, p.source, repositorycommon.ModuleName, nil, repositoryevents.FindSkillsCommand{Patterns: patterns, Repositories: names}))
	v, err := event.GetAs[repositoryevents.FindSkillsResult](r)
	if err != nil {
		return nil, err
	}
	return v.Skills, nil
}
func (p *eventHubRepository) LoadSkill(id, name string) (*spec.Skill, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicLoadSkill, p.source, repositorycommon.ModuleName, nil, repositoryevents.LoadSkillCommand{SkillID: id, RepoName: name}))
	v, err := event.GetAs[repositoryevents.LoadSkillResult](r)
	if err != nil {
		return nil, err
	}
	return &v.Skill, nil
}
func (p *eventHubRepository) CheckDefaultSkill(skillID string) (bool, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicCheckDefaultSkill, p.source, repositorycommon.ModuleName, nil, repositoryevents.CheckDefaultSkillCommand{SkillID: skillID}))
	v, err := event.GetAs[repositoryevents.CheckDefaultSkillResult](r)
	if err != nil {
		return false, err
	}
	return v.Exists, nil
}
func (p *eventHubRepository) RebuildIndex(name string) error {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicRebuildIndex, p.source, repositorycommon.ModuleName, nil, repositoryevents.RebuildIndexCommand{Repository: name}))
	_, err := event.GetAs[repositoryevents.RebuildIndexResult](r)
	if err != nil {
		return err
	}
	return nil
}
func (p *eventHubRepository) Archive(skillID, sourcePath string) error {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicArchiveSkill, p.source, repositorycommon.ModuleName, nil, repositoryevents.ArchiveSkillCommand{SkillID: skillID, SourcePath: sourcePath}))
	_, err := event.GetAs[repositoryevents.ArchiveSkillResult](r)
	if err != nil {
		return err
	}
	return nil
}
func (p *eventHubRepository) Add(repository config.RepositoryConfig) error {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicAddRepository, p.source, repositorycommon.ModuleName, nil, repositoryevents.AddRepositoryCommand{Repository: repository}))
	_, err := event.GetAs[repositoryevents.AddRepositoryResult](r)
	if err != nil {
		return err
	}
	return nil
}
func (p *eventHubRepository) Remove(name string) error {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicRemoveRepository, p.source, repositorycommon.ModuleName, nil, repositoryevents.RemoveRepositoryCommand{Name: name}))
	_, err := event.GetAs[repositoryevents.RemoveRepositoryResult](r)
	if err != nil {
		return err
	}
	return nil
}
func (p *eventHubRepository) Sync(name string) (*gitpkg.RepositorySyncResult, error) {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicSyncRepository, p.source, repositorycommon.ModuleName, nil, repositoryevents.SyncRepositoryCommand{Name: name}))
	v, err := event.GetAs[repositoryevents.SyncRepositoryResult](r)
	if err != nil {
		return nil, err
	}
	return &v.Result, nil
}
func (p *eventHubRepository) Enable(name string) error {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicEnableRepository, p.source, repositorycommon.ModuleName, nil, repositoryevents.EnableRepositoryCommand{Name: name}))
	_, err := event.GetAs[repositoryevents.EnableRepositoryResult](r)
	if err != nil {
		return err
	}
	return nil
}
func (p *eventHubRepository) Disable(name string) error {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicDisableRepository, p.source, repositorycommon.ModuleName, nil, repositoryevents.DisableRepositoryCommand{Name: name}))
	_, err := event.GetAs[repositoryevents.DisableRepositoryResult](r)
	if err != nil {
		return err
	}
	return nil
}
func (p *eventHubRepository) SetDefault(name string) error {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicSetDefault, p.source, repositorycommon.ModuleName, nil, repositoryevents.SetDefaultCommand{Name: name}))
	if r == nil {
		return fmt.Errorf("repository set-default returned no result")
	}
	_, err := event.GetAs[repositoryevents.SetDefaultResult](r)
	if err != nil {
		return err
	}
	return nil
}

func (p *eventHubRepository) UpdateURL(name, url string) error {
	r := p.hub.Send(event.NewEvent(repositoryevents.TopicUpdateURL, p.source, repositorycommon.ModuleName, nil, repositoryevents.UpdateURLCommand{Name: name, URL: url}))
	_, err := event.GetAs[repositoryevents.UpdateURLResult](r)
	if err != nil {
		return err
	}
	return nil
}
