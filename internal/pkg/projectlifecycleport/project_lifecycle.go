// Package projectlifecycleport exposes lifecycle DTOs without coupling
// inbound adapters to the application service package.
package projectlifecycleport

import projectlifecycle "github.com/muidea/skill-hub/internal/modules/application/project_lifecycle/service"

type RegisterResult = projectlifecycle.RegisterResult
type ImportOptions = projectlifecycle.ImportOptions
type ImportSummary = projectlifecycle.ImportSummary
type DedupeOptions = projectlifecycle.DedupeOptions
type DuplicateReport = projectlifecycle.DuplicateReport
type SyncCopiesOptions = projectlifecycle.SyncCopiesOptions
type SyncCopiesResult = projectlifecycle.SyncCopiesResult
type PathLintOptions = projectlifecycle.PathLintOptions
type PathLintReport = projectlifecycle.PathLintReport
type ValidateOptions = projectlifecycle.ValidateOptions
type ValidateReport = projectlifecycle.ValidateReport
type AuditOptions = projectlifecycle.AuditOptions
type AuditReport = projectlifecycle.AuditReport
