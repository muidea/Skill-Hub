// Package events owns typed commands for the repository Block.
package events

import (
	"github.com/muidea/skill-hub/internal/config"
	gitpkg "github.com/muidea/skill-hub/internal/git"
	"github.com/muidea/skill-hub/pkg/spec"
)

const (
	TopicDefaultRepository = "skillhub.repository.default"
	TopicGetRepository     = "skillhub.repository.get"
	TopicRepositoryPath    = "skillhub.repository.path"
	TopicArchiveSkill      = "skillhub.repository.archive-skill"
	TopicCheckDefaultSkill = "skillhub.repository.check-default-skill"
	TopicFindSkill         = "skillhub.repository.find-skill"
	TopicLoadSkill         = "skillhub.repository.load-skill"
	TopicRebuildIndex      = "skillhub.repository.rebuild-index"
	TopicListRepositories  = "skillhub.repository.list"
	TopicAddRepository     = "skillhub.repository.add"
	TopicRemoveRepository  = "skillhub.repository.remove"
	TopicSyncRepository    = "skillhub.repository.sync"
	TopicEnableRepository  = "skillhub.repository.enable"
	TopicDisableRepository = "skillhub.repository.disable"
	TopicSetDefault        = "skillhub.repository.set-default"
	TopicListSkills        = "skillhub.repository.list-skills"
	TopicFindSkills        = "skillhub.repository.find-skills"
	TopicUpdateURL         = "skillhub.repository.update-url"
)

type DefaultRepositoryCommand struct{}
type DefaultRepositoryResult struct{ Repository config.RepositoryConfig }
type GetRepositoryCommand struct{ Name string }
type GetRepositoryResult struct{ Repository config.RepositoryConfig }
type RepositoryPathCommand struct{ Name string }
type RepositoryPathResult struct{ Path string }

type ArchiveSkillCommand struct {
	SkillID    string
	SourcePath string
}
type ArchiveSkillResult struct{}

type CheckDefaultSkillCommand struct{ SkillID string }
type CheckDefaultSkillResult struct{ Exists bool }

type FindSkillCommand struct{ SkillID string }
type FindSkillResult struct{ Skills []spec.SkillMetadata }

type LoadSkillCommand struct {
	SkillID  string
	RepoName string
}
type LoadSkillResult struct{ Skill spec.Skill }

type RebuildIndexCommand struct{ Repository string }
type RebuildIndexResult struct{}

type ListRepositoriesCommand struct{ IncludeDisabled bool }
type ListRepositoriesResult struct{ Repositories []config.RepositoryConfig }
type AddRepositoryCommand struct{ Repository config.RepositoryConfig }
type AddRepositoryResult struct{}
type RemoveRepositoryCommand struct{ Name string }
type RemoveRepositoryResult struct{}
type SyncRepositoryCommand struct{ Name string }
type SyncRepositoryResult struct{ Result gitpkg.RepositorySyncResult }
type EnableRepositoryCommand struct{ Name string }
type EnableRepositoryResult struct{}
type DisableRepositoryCommand struct{ Name string }
type DisableRepositoryResult struct{}
type SetDefaultCommand struct{ Name string }
type SetDefaultResult struct{}
type ListSkillsCommand struct{ Repositories []string }
type ListSkillsResult struct{ Skills []spec.SkillMetadata }
type FindSkillsCommand struct {
	Patterns     []string
	Repositories []string
}
type FindSkillsResult struct{ Skills []spec.SkillMetadata }
type UpdateURLCommand struct {
	Name string
	URL  string
}
type UpdateURLResult struct{}
