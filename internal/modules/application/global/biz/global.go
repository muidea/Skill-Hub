package biz

import (
	globalservice "github.com/muidea/skill-hub/internal/modules/application/global/service"
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

const ownerID = "global_skill"

// Global owns cross-agent global-skill orchestration. Repository access is
// expressed only through the repository owner's typed EventHub port.
type Global struct {
	basebiz.Base
	service *globalservice.Global
}

func New(hub event.Hub, background task.BackgroundRoutine) *Global {
	return &Global{
		Base:    basebiz.New(ownerID, hub, background),
		service: globalservice.New(repositoryport.NewProjectSource(hub, ownerID)),
	}
}

func (b *Global) Service() *globalservice.Global { return b.service }
func (b *Global) Teardown()                      {}
