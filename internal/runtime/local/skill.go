package local

import (
	"github.com/muidea/skill-hub/internal/adapters/remotesearch"
	"github.com/muidea/skill-hub/internal/engine"
	"github.com/muidea/skill-hub/internal/pkg/skillport"
	"github.com/muidea/skill-hub/pkg/spec"
)

type Skill struct{}

var _ skillport.Skill = (*Skill)(nil)

func NewSkill() *Skill                      { return &Skill{} }
func (s *Skill) SkillsDir() (string, error) { return engine.GetSkillsDir() }
func (s *Skill) SearchRemote(keyword string, limit int) ([]spec.RemoteSearchResult, error) {
	return remotesearch.Search(keyword, limit)
}
