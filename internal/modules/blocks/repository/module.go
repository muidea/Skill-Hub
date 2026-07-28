package repository

import (
	"context"

	"github.com/muidea/skill-hub/internal/modules/blocks/repository/biz"
	"github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/common"
	"github.com/muidea/skill-hub/internal/modules/blocks/repository/service"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	pluginmodule "github.com/muidea/magicCommon/framework/plugin/module"
	"github.com/muidea/magicCommon/task"
)

func init() { pluginmodule.Register(New()) }

type Repository struct {
	bizPtr     *biz.Repository
	servicePtr *service.Repository
}

func New() *Repository {
	return &Repository{
		servicePtr: service.New(),
	}
}

func (r *Repository) ID() string  { return common.ModuleName }
func (r *Repository) Weight() int { return 10 }
func (r *Repository) Setup(_ context.Context, hub event.Hub, background task.BackgroundRoutine) *cd.Error {
	r.bizPtr = biz.New(hub, background, r.servicePtr)
	return nil
}
func (r *Repository) Run(context.Context) *cd.Error { return nil }
func (r *Repository) Teardown(context.Context) {
	if r.bizPtr != nil {
		r.bizPtr.Teardown()
		r.bizPtr = nil
	}
}

func (r *Repository) Service() *service.Repository {
	return r.servicePtr
}
