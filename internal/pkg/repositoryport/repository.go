// Package repositoryport defines the local repository capability consumed by
// command and protocol adapters. It deliberately exposes data operations,
// never repository service implementations.
package repositoryport

import (
	"github.com/muidea/skill-hub/internal/config"
	gitpkg "github.com/muidea/skill-hub/internal/git"
	"github.com/muidea/skill-hub/pkg/spec"
)

// RepositoryConfig is the stable repository configuration value shared with
// command and protocol adapters. Its persistence representation remains owned
// by the repository capability.
type RepositoryConfig = config.RepositoryConfig

// Repository is the narrow boundary for repository configuration, discovery
// and skill-source operations in the embedded runtime.
type Repository interface {
	Default() (*config.RepositoryConfig, error)
	Get(name string) (*config.RepositoryConfig, error)
	Path(name string) (string, error)
	List(includeDisabled bool) ([]config.RepositoryConfig, error)
	ListSkills(repoNames []string) ([]spec.SkillMetadata, error)
	FindSkill(skillID string) ([]spec.SkillMetadata, error)
	FindSkills(patterns, repoNames []string) ([]spec.SkillMetadata, error)
	LoadSkill(skillID, repoName string) (*spec.Skill, error)
	CheckDefaultSkill(skillID string) (bool, error)
	RebuildIndex(name string) error
	Archive(skillID, sourcePath string) error
	Add(config.RepositoryConfig) error
	Remove(name string) error
	Sync(name string) (*gitpkg.RepositorySyncResult, error)
	Enable(name string) error
	Disable(name string) error
	SetDefault(name string) error
	UpdateURL(name, url string) error
}
