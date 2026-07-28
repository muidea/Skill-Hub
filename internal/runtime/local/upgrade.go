package local

import (
	"context"

	upgradeservice "github.com/muidea/skill-hub/internal/modules/application/upgrade/service"
	"github.com/muidea/skill-hub/internal/pkg/upgradeport"
)

// Upgrade is the embedded adapter for the upgrade use case.
type Upgrade struct{ service *upgradeservice.Service }

func NewUpgrade() *Upgrade { return &Upgrade{service: upgradeservice.New()} }
func (u *Upgrade) Check(ctx context.Context, opts upgradeport.Options) (*upgradeport.Result, error) {
	return u.service.Check(ctx, opts)
}
func (u *Upgrade) Apply(ctx context.Context, opts upgradeport.Options) (*upgradeport.Result, error) {
	return u.service.Upgrade(ctx, opts)
}
