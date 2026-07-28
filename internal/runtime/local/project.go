package local

import (
	projectapplymodule "github.com/muidea/skill-hub/internal/modules/application/project_apply"
	projectapplyservice "github.com/muidea/skill-hub/internal/modules/application/project_apply/service"
	projectfeedbackmodule "github.com/muidea/skill-hub/internal/modules/application/project_feedback"
	projectfeedbackservice "github.com/muidea/skill-hub/internal/modules/application/project_feedback/service"
	projectinventorymodule "github.com/muidea/skill-hub/internal/modules/application/project_inventory"
	projectinventoryservice "github.com/muidea/skill-hub/internal/modules/application/project_inventory/service"
	projectlifecyclemodule "github.com/muidea/skill-hub/internal/modules/application/project_lifecycle"
	projectlifecycleservice "github.com/muidea/skill-hub/internal/modules/application/project_lifecycle/service"
	projectstatusmodule "github.com/muidea/skill-hub/internal/modules/application/project_status"
	projectstatusservice "github.com/muidea/skill-hub/internal/modules/application/project_status/service"
	projectusemodule "github.com/muidea/skill-hub/internal/modules/application/project_use"
	projectuseservice "github.com/muidea/skill-hub/internal/modules/application/project_use/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

// Project groups project-owned use cases for the embedded runtime. It does
// not expose a generic GetService API or other owner implementations.
type Project struct {
	apply     *projectapplymodule.ProjectApply
	feedback  *projectfeedbackmodule.ProjectFeedback
	inventory *projectinventorymodule.ProjectInventory
	lifecycle *projectlifecyclemodule.ProjectLifecycle
	status    *projectstatusmodule.ProjectStatus
	use       *projectusemodule.ProjectUse
}

func NewProject(hub event.Hub, background task.BackgroundRoutine) *Project {
	return &Project{apply: projectapplymodule.New(hub, background), feedback: projectfeedbackmodule.New(hub, background), inventory: projectinventorymodule.New(hub, background), lifecycle: projectlifecyclemodule.New(hub, background), status: projectstatusmodule.New(hub, background), use: projectusemodule.New(hub, background)}
}

func (p *Project) ListProjects() ([]projectinventoryservice.ProjectSummary, error) {
	return p.inventory.Service().ListProjects()
}
func (p *Project) GetProject(id string) (*projectinventoryservice.ProjectDetail, error) {
	return p.inventory.Service().GetProject(id)
}
func (p *Project) ListProjectSkills(id string) ([]projectinventoryservice.ProjectSkill, error) {
	return p.inventory.Service().ListProjectSkills(id)
}
func (p *Project) Inspect(projectPath, skillID string) (*projectstatusservice.ProjectStatusSummary, error) {
	return p.status.Service().Inspect(projectPath, skillID)
}
func (p *Project) Enable(projectPath, skillID, repoName string, variables map[string]string) (*projectuseservice.UseResult, error) {
	return p.use.Service().EnableSkill(projectPath, skillID, repoName, variables)
}
func (p *Project) Apply(projectPath, skillID string, dryRun, force bool) (*projectapplyservice.ApplyResult, error) {
	return p.apply.Service().Apply(projectPath, skillID, dryRun, force)
}
func (p *Project) PreviewFeedback(projectPath, skillID string) (*projectfeedbackservice.PreviewResult, error) {
	return p.feedback.Service().Preview(projectPath, skillID)
}
func (p *Project) ApplyFeedback(projectPath, skillID string) (*projectfeedbackservice.PreviewResult, error) {
	return p.feedback.Service().Apply(projectPath, skillID)
}
func (p *Project) Register(projectPath, skillID string, skipValidate bool) (*projectlifecycleservice.RegisterResult, error) {
	return p.lifecycle.Service().Register(projectPath, skillID, skipValidate)
}
func (p *Project) Import(projectPath, skillsDir string, opts projectlifecycleservice.ImportOptions) (*projectlifecycleservice.ImportSummary, error) {
	return p.lifecycle.Service().Import(projectPath, skillsDir, opts)
}
func (p *Project) Dedupe(scope string, opts projectlifecycleservice.DedupeOptions) (*projectlifecycleservice.DuplicateReport, error) {
	return p.lifecycle.Service().Dedupe(scope, opts)
}
func (p *Project) SyncCopies(opts projectlifecycleservice.SyncCopiesOptions) (*projectlifecycleservice.SyncCopiesResult, error) {
	return p.lifecycle.Service().SyncCopies(opts)
}
func (p *Project) LintPaths(opts projectlifecycleservice.PathLintOptions) (*projectlifecycleservice.PathLintReport, error) {
	return p.lifecycle.Service().LintPaths(opts)
}
func (p *Project) Validate(opts projectlifecycleservice.ValidateOptions) (*projectlifecycleservice.ValidateReport, error) {
	return p.lifecycle.Service().ValidateProjectSkills(opts)
}
func (p *Project) Audit(opts projectlifecycleservice.AuditOptions) (*projectlifecycleservice.AuditReport, error) {
	return p.lifecycle.Service().Audit(opts)
}

func (p *Project) Close() {
	if p != nil && p.apply != nil {
		p.apply.Teardown()
		p.apply = nil
	}
	if p != nil && p.feedback != nil {
		p.feedback.Teardown()
		p.feedback = nil
	}
	if p != nil && p.use != nil {
		p.use.Teardown()
		p.use = nil
	}
	if p != nil && p.inventory != nil {
		p.inventory.Teardown()
		p.inventory = nil
	}
	if p != nil && p.status != nil {
		p.status.Teardown()
		p.status = nil
	}
	if p != nil && p.lifecycle != nil {
		p.lifecycle.Teardown()
		p.lifecycle = nil
	}
}
