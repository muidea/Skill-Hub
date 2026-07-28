package biz

import (
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/common"
	"github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/events"
	"github.com/muidea/skill-hub/internal/modules/blocks/repository/service"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type Repository struct {
	basebiz.Base
	service *service.Repository
}

func New(hub event.Hub, background task.BackgroundRoutine, repositoryService *service.Repository) *Repository {
	biz := &Repository{Base: basebiz.New(common.ModuleName, hub, background), service: repositoryService}
	biz.SubscribeFunc(events.TopicDefaultRepository, biz.handleDefaultRepository)
	biz.SubscribeFunc(events.TopicGetRepository, biz.handleGetRepository)
	biz.SubscribeFunc(events.TopicRepositoryPath, biz.handleRepositoryPath)
	biz.SubscribeFunc(events.TopicArchiveSkill, biz.handleArchiveSkill)
	biz.SubscribeFunc(events.TopicCheckDefaultSkill, biz.handleCheckDefaultSkill)
	biz.SubscribeFunc(events.TopicFindSkill, biz.handleFindSkill)
	biz.SubscribeFunc(events.TopicLoadSkill, biz.handleLoadSkill)
	biz.SubscribeFunc(events.TopicRebuildIndex, biz.handleRebuildIndex)
	biz.SubscribeFunc(events.TopicListRepositories, biz.handleListRepositories)
	biz.SubscribeFunc(events.TopicAddRepository, biz.handleAddRepository)
	biz.SubscribeFunc(events.TopicRemoveRepository, biz.handleRemoveRepository)
	biz.SubscribeFunc(events.TopicSyncRepository, biz.handleSyncRepository)
	biz.SubscribeFunc(events.TopicEnableRepository, biz.handleEnableRepository)
	biz.SubscribeFunc(events.TopicDisableRepository, biz.handleDisableRepository)
	biz.SubscribeFunc(events.TopicSetDefault, biz.handleSetDefault)
	biz.SubscribeFunc(events.TopicListSkills, biz.handleListSkills)
	biz.SubscribeFunc(events.TopicFindSkills, biz.handleFindSkills)
	biz.SubscribeFunc(events.TopicUpdateURL, biz.handleUpdateURL)
	return biz
}

func (b *Repository) Teardown() {
	b.UnsubscribeFunc(events.TopicDefaultRepository)
	b.UnsubscribeFunc(events.TopicGetRepository)
	b.UnsubscribeFunc(events.TopicRepositoryPath)
	b.UnsubscribeFunc(events.TopicArchiveSkill)
	b.UnsubscribeFunc(events.TopicCheckDefaultSkill)
	b.UnsubscribeFunc(events.TopicFindSkill)
	b.UnsubscribeFunc(events.TopicLoadSkill)
	b.UnsubscribeFunc(events.TopicRebuildIndex)
	b.UnsubscribeFunc(events.TopicListRepositories)
	b.UnsubscribeFunc(events.TopicAddRepository)
	b.UnsubscribeFunc(events.TopicRemoveRepository)
	b.UnsubscribeFunc(events.TopicSyncRepository)
	b.UnsubscribeFunc(events.TopicEnableRepository)
	b.UnsubscribeFunc(events.TopicDisableRepository)
	b.UnsubscribeFunc(events.TopicSetDefault)
	b.UnsubscribeFunc(events.TopicListSkills)
	b.UnsubscribeFunc(events.TopicFindSkills)
	b.UnsubscribeFunc(events.TopicUpdateURL)
}

func (b *Repository) handleDefaultRepository(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	if _, ok := e.Data().(events.DefaultRepositoryCommand); !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository default command"))
		return
	}
	repository, err := b.service.DefaultRepository()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	if repository == nil {
		result.Set(nil, cd.NewError(cd.Unexpected, "default repository is unavailable"))
		return
	}
	result.Set(events.DefaultRepositoryResult{Repository: *repository}, nil)
}

func (b *Repository) handleGetRepository(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.GetRepositoryCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository get command"))
		return
	}
	repository, err := b.service.GetRepository(command.Name)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	if repository == nil {
		result.Set(nil, cd.NewError(cd.Unexpected, "repository is unavailable"))
		return
	}
	result.Set(events.GetRepositoryResult{Repository: *repository}, nil)
}

func (b *Repository) handleRepositoryPath(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.RepositoryPathCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository path command"))
		return
	}
	path, err := b.service.Path(command.Name)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.RepositoryPathResult{Path: path}, nil)
}

func (b *Repository) handleArchiveSkill(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.ArchiveSkillCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository archive command"))
		return
	}
	if err := b.service.ArchiveToDefaultRepository(command.SkillID, command.SourcePath); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.ArchiveSkillResult{}, nil)
}

func (b *Repository) handleCheckDefaultSkill(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.CheckDefaultSkillCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository check-default-skill command"))
		return
	}
	exists, err := b.service.CheckSkillInDefaultRepository(command.SkillID)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.CheckDefaultSkillResult{Exists: exists}, nil)
}

func (b *Repository) handleFindSkill(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.FindSkillCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository find command"))
		return
	}
	skills, err := b.service.FindSkill(command.SkillID)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.FindSkillResult{Skills: skills}, nil)
}

func (b *Repository) handleLoadSkill(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.LoadSkillCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository load command"))
		return
	}
	skill, err := b.service.LoadSkill(command.SkillID, command.RepoName)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	if skill == nil {
		result.Set(nil, cd.NewError(cd.Unexpected, "repository skill is unavailable"))
		return
	}
	result.Set(events.LoadSkillResult{Skill: *skill}, nil)
}

func (b *Repository) handleRebuildIndex(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.RebuildIndexCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository rebuild-index command"))
		return
	}
	if err := b.service.RebuildRepositoryIndex(command.Repository); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.RebuildIndexResult{}, nil)
}

func (b *Repository) handleListRepositories(e event.Event, result event.Result) {
	command, ok := e.Data().(events.ListRepositoriesCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository list command"))
		return
	}
	items, err := b.service.ListRepositories(command.IncludeDisabled)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.ListRepositoriesResult{Repositories: items}, nil)
}
func (b *Repository) handleAddRepository(e event.Event, result event.Result) {
	command, ok := e.Data().(events.AddRepositoryCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository add command"))
		return
	}
	if err := b.service.AddRepository(command.Repository); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.AddRepositoryResult{}, nil)
}
func (b *Repository) handleRemoveRepository(e event.Event, result event.Result) {
	command, ok := e.Data().(events.RemoveRepositoryCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository remove command"))
		return
	}
	if err := b.service.RemoveRepository(command.Name); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.RemoveRepositoryResult{}, nil)
}
func (b *Repository) handleSyncRepository(e event.Event, result event.Result) {
	command, ok := e.Data().(events.SyncRepositoryCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository sync command"))
		return
	}
	value, err := b.service.SyncRepository(command.Name)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	if value == nil {
		result.Set(nil, cd.NewError(cd.Unexpected, "repository sync result is unavailable"))
		return
	}
	result.Set(events.SyncRepositoryResult{Result: *value}, nil)
}
func (b *Repository) handleEnableRepository(e event.Event, result event.Result) {
	command, ok := e.Data().(events.EnableRepositoryCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository enable command"))
		return
	}
	if err := b.service.EnableRepository(command.Name); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.EnableRepositoryResult{}, nil)
}
func (b *Repository) handleDisableRepository(e event.Event, result event.Result) {
	command, ok := e.Data().(events.DisableRepositoryCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository disable command"))
		return
	}
	if err := b.service.DisableRepository(command.Name); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.DisableRepositoryResult{}, nil)
}
func (b *Repository) handleSetDefault(e event.Event, result event.Result) {
	command, ok := e.Data().(events.SetDefaultCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository set-default command"))
		return
	}
	if err := b.service.SetDefaultRepository(command.Name); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.SetDefaultResult{}, nil)
}
func (b *Repository) handleListSkills(e event.Event, result event.Result) {
	command, ok := e.Data().(events.ListSkillsCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository list-skills command"))
		return
	}
	items, err := b.service.ListSkills(command.Repositories)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.ListSkillsResult{Skills: items}, nil)
}
func (b *Repository) handleFindSkills(e event.Event, result event.Result) {
	command, ok := e.Data().(events.FindSkillsCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository find-skills command"))
		return
	}
	items, err := b.service.FindSkillsByPatterns(command.Patterns, command.Repositories)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.FindSkillsResult{Skills: items}, nil)
}

func (b *Repository) handleUpdateURL(e event.Event, result event.Result) {
	command, ok := e.Data().(events.UpdateURLCommand)
	if result == nil {
		return
	}
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid repository update-url command"))
		return
	}
	if err := b.service.UpdateRepositoryURL(command.Name, command.URL); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.UpdateURLResult{}, nil)
}
