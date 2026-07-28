// Package gitport defines the Git operations needed by local command and API
// adapters without exposing the Git Block implementation.
package gitport

import gitpkg "github.com/muidea/skill-hub/internal/git"

type RemoteUpdateStatus = gitpkg.RemoteUpdateStatus

type Git interface {
	CloneRepository(path, url string) error
	InitializeRepository(path string) error
	RepositoryRemoteURLs(path string) ([]string, error)
	SyncDefaultRepository(force bool) (int, error)
	SyncSkillRepositoryAndRefresh() error
	CheckSkillRepositoryUpdates() (*gitpkg.RemoteUpdateStatus, error)
	SkillRepositoryStatus() (string, error)
	PushSkillRepositoryChanges(message string) error
	PushSkillRepositoryCommits() error
	SetSkillRepositoryRemote(url string) error
}

func SuggestedCommitMessage(files []string) string { return gitpkg.SuggestedCommitMessage(files) }
func SuggestedCommitMessageFromStatus(status string) string {
	return gitpkg.SuggestedCommitMessageFromStatus(status)
}
