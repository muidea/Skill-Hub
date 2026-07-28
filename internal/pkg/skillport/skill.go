// Package skillport defines owner-neutral skill lookup operations used by the
// local runtime's command and HTTP adapters.
package skillport

import "github.com/muidea/skill-hub/pkg/spec"

type Skill interface {
	SkillsDir() (string, error)
	SearchRemote(keyword string, limit int) ([]spec.RemoteSearchResult, error)
}
