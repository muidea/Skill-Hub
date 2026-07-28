// Package projectport exposes the project use-case DTO surface consumed by
// CLI adapters. It keeps protocol rendering independent from service packages.
package projectport

type ApplyResult struct {
	ProjectPath string            `json:"project_path"`
	DryRun      bool              `json:"dry_run"`
	Items       []ApplyResultItem `json:"items"`
}

type ApplyResultItem struct {
	SkillID   string `json:"skill_id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Variables int    `json:"variables"`
}

type PreviewResult struct {
	ProjectPath       string   `json:"project_path"`
	SkillID           string   `json:"skill_id"`
	DefaultRepo       string   `json:"default_repo"`
	SkillExists       bool     `json:"skill_exists"`
	Changes           []string `json:"changes"`
	HasContentChanges bool     `json:"has_content_changes"`
	ProjectVersion    string   `json:"project_version"`
	RepoVersion       string   `json:"repo_version"`
	ResolvedVersion   string   `json:"resolved_version"`
	NeedsVersionBump  bool     `json:"needs_version_bump"`
	Blocked           bool     `json:"blocked"`
	BlockReason       string   `json:"block_reason,omitempty"`
	NoChanges         bool     `json:"no_changes"`
}

type ProjectStatusSummary struct {
	ProjectPath string            `json:"project_path"`
	Scope       string            `json:"scope"`
	SkillCount  int               `json:"skill_count"`
	Items       []SkillStatusItem `json:"items"`
}

type SkillStatusItem struct {
	SkillID          string `json:"skill_id"`
	Status           string `json:"status"`
	LegacyStatus     string `json:"legacy_status,omitempty"`
	Reason           string `json:"reason,omitempty"`
	SourceRepository string `json:"source_repository,omitempty"`
	LocalVersion     string `json:"local_version,omitempty"`
	RepoVersion      string `json:"repo_version,omitempty"`
	LocalPath        string `json:"local_path,omitempty"`
	RepoPath         string `json:"repo_path,omitempty"`
}
