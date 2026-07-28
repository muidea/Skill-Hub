// Package globalport exposes global skill result DTOs to protocol adapters.
package globalport

import global "github.com/muidea/skill-hub/internal/modules/application/global/service"

type StatusSummary = global.StatusSummary
type StatusItem = global.StatusItem
type ApplyResult = global.ApplyResult
type RemoveResult = global.RemoveResult

const (
	StatusPlanned         = global.StatusPlanned
	StatusApplied         = global.StatusApplied
	StatusConflict        = global.StatusConflict
	StatusError           = global.StatusError
	StatusRemoved         = global.StatusRemoved
	StatusNotApplied      = global.StatusNotApplied
	StatusMissingAgentDir = global.StatusMissingAgentDir
	StatusOK              = global.StatusOK
	StatusStale           = global.StatusStale
	StatusModified        = global.StatusModified
	StatusOrphaned        = global.StatusOrphaned
)
