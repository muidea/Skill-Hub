package local

import (
	globalmodule "github.com/muidea/skill-hub/internal/modules/application/global"
	globalservice "github.com/muidea/skill-hub/internal/modules/application/global/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

// Global owns the embedded adapter for machine-global skill state.
type Global struct{ module *globalmodule.Global }

func NewGlobal(hub event.Hub, background task.BackgroundRoutine) *Global {
	return &Global{module: globalmodule.New(hub, background)}
}
func (g *Global) Inspect(skillID string, agents []string) (*globalservice.StatusSummary, error) {
	return g.module.Service().Inspect(skillID, agents)
}
func (g *Global) Enable(skillID, repository string, agents []string, variables map[string]string) (*globalservice.UseResult, error) {
	return g.module.Service().EnableSkill(skillID, repository, agents, variables)
}
func (g *Global) Apply(skillID string, agents []string, dryRun, force bool) (*globalservice.ApplyResult, error) {
	return g.module.Service().Apply(skillID, agents, dryRun, force)
}
func (g *Global) Remove(skillID string, agents []string, force bool) (*globalservice.RemoveResult, error) {
	return g.module.Service().Remove(skillID, agents, force)
}

func (g *Global) Close() {
	if g != nil && g.module != nil {
		g.module.Teardown()
		g.module = nil
	}
}
