// Package remote implements the daemon-backed execution mode for skill-hub.
// It owns service discovery and the HTTP protocol client; callers only depend
// on the typed command/result contract exposed by Client.
package remote

import (
	"context"
	"os"
	"strings"
	"time"

	hubclientmodule "github.com/muidea/skill-hub/internal/adapters/hubclient"
	"github.com/muidea/skill-hub/internal/config"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/pkg/spec"
)

const defaultServiceURL = "http://127.0.0.1:5525"

type serviceBridgeClient interface {
	Available(ctx context.Context) bool
	ListRepos(ctx context.Context) (*httpapibiz.RepoListData, error)
	AddRepo(ctx context.Context, repo config.RepositoryConfig) error
	RemoveRepo(ctx context.Context, name string) error
	SyncRepo(ctx context.Context, name string) (*httpapibiz.RepoSyncData, error)
	EnableRepo(ctx context.Context, name string) error
	DisableRepo(ctx context.Context, name string) error
	SetDefaultRepo(ctx context.Context, name string) error
	SkillRepositoryStatus(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error)
	CheckSkillRepositoryUpdates(ctx context.Context) (*httpapibiz.SkillRepositoryCheckData, error)
	SyncSkillRepositoryAndRefresh(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error)
	PushSkillRepositoryPreview(ctx context.Context) (*httpapibiz.PushSkillRepositoryPreviewData, error)
	PushSkillRepositoryChanges(ctx context.Context, req httpapibiz.PushSkillRepositoryRequest) (*httpapibiz.PushSkillRepositoryData, error)
	ListSkills(ctx context.Context, repoNames []string) ([]spec.SkillMetadata, error)
	FindSkillsByPatterns(ctx context.Context, patterns, repoNames []string) ([]spec.SkillMetadata, error)
	SearchRemoteSkills(ctx context.Context, keyword string, limit int) ([]spec.RemoteSearchResult, error)
	GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error)
	FindSkillCandidates(ctx context.Context, skillID string) ([]spec.SkillMetadata, error)
	GetSkillDetail(ctx context.Context, skillID, repoName string) (*spec.Skill, error)
	UseSkill(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error)
	UseGlobalSkill(ctx context.Context, req httpapibiz.UseGlobalSkillRequest) (*httpapibiz.UseGlobalSkillData, error)
	RegisterSkill(ctx context.Context, req httpapibiz.RegisterSkillRequest) (*httpapibiz.RegisterSkillData, error)
	ImportSkills(ctx context.Context, req httpapibiz.ImportSkillsRequest) (*httpapibiz.ImportSkillsData, error)
	DedupeSkills(ctx context.Context, req httpapibiz.DedupeRequest) (*httpapibiz.DedupeData, error)
	SyncCopies(ctx context.Context, req httpapibiz.SyncCopiesRequest) (*httpapibiz.SyncCopiesData, error)
	LintPaths(ctx context.Context, req httpapibiz.PathLintRequest) (*httpapibiz.PathLintData, error)
	ValidateProjectSkills(ctx context.Context, req httpapibiz.ValidateProjectSkillsRequest) (*httpapibiz.ValidateProjectSkillsData, error)
	AuditProjectSkills(ctx context.Context, req httpapibiz.AuditProjectSkillsRequest) (*httpapibiz.AuditProjectSkillsData, error)
	ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error)
	GetGlobalStatus(ctx context.Context, skillID string, agents []string) (*httpapibiz.GlobalStatusData, error)
	ApplyGlobal(ctx context.Context, req httpapibiz.ApplyGlobalRequest) (*httpapibiz.ApplyGlobalData, error)
	RemoveGlobalSkill(ctx context.Context, skillID string, agents []string, force bool) (*httpapibiz.RemoveGlobalSkillData, error)
	PreviewFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
	ApplyFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
}

// Client is the typed remote execution contract used by the CLI runtime
// selector. It intentionally contains no daemon implementation details.
type Client = serviceBridgeClient

type hubBridgeClient struct {
	client *hubclientmodule.HubClient
}

func (h *hubBridgeClient) Available(ctx context.Context) bool {
	return h.client.Available(ctx)
}

func (h *hubBridgeClient) ListRepos(ctx context.Context) (*httpapibiz.RepoListData, error) {
	return h.client.ListRepos(ctx)
}

func (h *hubBridgeClient) AddRepo(ctx context.Context, repo config.RepositoryConfig) error {
	return h.client.AddRepo(ctx, repo)
}

func (h *hubBridgeClient) RemoveRepo(ctx context.Context, name string) error {
	return h.client.RemoveRepo(ctx, name)
}

func (h *hubBridgeClient) SyncRepo(ctx context.Context, name string) (*httpapibiz.RepoSyncData, error) {
	return h.client.SyncRepo(ctx, name)
}

func (h *hubBridgeClient) EnableRepo(ctx context.Context, name string) error {
	return h.client.EnableRepo(ctx, name)
}

func (h *hubBridgeClient) DisableRepo(ctx context.Context, name string) error {
	return h.client.DisableRepo(ctx, name)
}

func (h *hubBridgeClient) SetDefaultRepo(ctx context.Context, name string) error {
	return h.client.SetDefaultRepo(ctx, name)
}

func (h *hubBridgeClient) SkillRepositoryStatus(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error) {
	return h.client.SkillRepositoryStatus(ctx)
}

func (h *hubBridgeClient) CheckSkillRepositoryUpdates(ctx context.Context) (*httpapibiz.SkillRepositoryCheckData, error) {
	return h.client.CheckSkillRepositoryUpdates(ctx)
}

func (h *hubBridgeClient) SyncSkillRepositoryAndRefresh(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error) {
	return h.client.SyncSkillRepositoryAndRefresh(ctx)
}

func (h *hubBridgeClient) PushSkillRepositoryPreview(ctx context.Context) (*httpapibiz.PushSkillRepositoryPreviewData, error) {
	return h.client.PushSkillRepositoryPreview(ctx)
}

func (h *hubBridgeClient) PushSkillRepositoryChanges(ctx context.Context, req httpapibiz.PushSkillRepositoryRequest) (*httpapibiz.PushSkillRepositoryData, error) {
	return h.client.PushSkillRepositoryChanges(ctx, req)
}

func (h *hubBridgeClient) ListSkills(ctx context.Context, repoNames []string) ([]spec.SkillMetadata, error) {
	return h.client.ListSkills(ctx, repoNames)
}

func (h *hubBridgeClient) FindSkillsByPatterns(ctx context.Context, patterns, repoNames []string) ([]spec.SkillMetadata, error) {
	return h.client.FindSkillsByPatterns(ctx, patterns, repoNames)
}

func (h *hubBridgeClient) SearchRemoteSkills(ctx context.Context, keyword string, limit int) ([]spec.RemoteSearchResult, error) {
	return h.client.SearchRemoteSkills(ctx, keyword, limit)
}

func (h *hubBridgeClient) GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error) {
	return h.client.GetProjectStatus(ctx, projectPath, skillID)
}

func (h *hubBridgeClient) FindSkillCandidates(ctx context.Context, skillID string) ([]spec.SkillMetadata, error) {
	return h.client.FindSkillCandidates(ctx, skillID)
}

func (h *hubBridgeClient) GetSkillDetail(ctx context.Context, skillID, repoName string) (*spec.Skill, error) {
	return h.client.GetSkillDetail(ctx, skillID, repoName)
}

func (h *hubBridgeClient) UseSkill(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error) {
	return h.client.UseSkill(ctx, req)
}

func (h *hubBridgeClient) UseGlobalSkill(ctx context.Context, req httpapibiz.UseGlobalSkillRequest) (*httpapibiz.UseGlobalSkillData, error) {
	return h.client.UseGlobalSkill(ctx, req)
}

func (h *hubBridgeClient) RegisterSkill(ctx context.Context, req httpapibiz.RegisterSkillRequest) (*httpapibiz.RegisterSkillData, error) {
	return h.client.RegisterSkill(ctx, req)
}

func (h *hubBridgeClient) ImportSkills(ctx context.Context, req httpapibiz.ImportSkillsRequest) (*httpapibiz.ImportSkillsData, error) {
	return h.client.ImportSkills(ctx, req)
}

func (h *hubBridgeClient) DedupeSkills(ctx context.Context, req httpapibiz.DedupeRequest) (*httpapibiz.DedupeData, error) {
	return h.client.DedupeSkills(ctx, req)
}

func (h *hubBridgeClient) SyncCopies(ctx context.Context, req httpapibiz.SyncCopiesRequest) (*httpapibiz.SyncCopiesData, error) {
	return h.client.SyncCopies(ctx, req)
}

func (h *hubBridgeClient) LintPaths(ctx context.Context, req httpapibiz.PathLintRequest) (*httpapibiz.PathLintData, error) {
	return h.client.LintPaths(ctx, req)
}

func (h *hubBridgeClient) ValidateProjectSkills(ctx context.Context, req httpapibiz.ValidateProjectSkillsRequest) (*httpapibiz.ValidateProjectSkillsData, error) {
	return h.client.ValidateProjectSkills(ctx, req)
}

func (h *hubBridgeClient) AuditProjectSkills(ctx context.Context, req httpapibiz.AuditProjectSkillsRequest) (*httpapibiz.AuditProjectSkillsData, error) {
	return h.client.AuditProjectSkills(ctx, req)
}

func (h *hubBridgeClient) ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error) {
	return h.client.ApplyProject(ctx, req)
}

func (h *hubBridgeClient) GetGlobalStatus(ctx context.Context, skillID string, agents []string) (*httpapibiz.GlobalStatusData, error) {
	return h.client.GetGlobalStatus(ctx, skillID, agents)
}

func (h *hubBridgeClient) ApplyGlobal(ctx context.Context, req httpapibiz.ApplyGlobalRequest) (*httpapibiz.ApplyGlobalData, error) {
	return h.client.ApplyGlobal(ctx, req)
}

func (h *hubBridgeClient) RemoveGlobalSkill(ctx context.Context, skillID string, agents []string, force bool) (*httpapibiz.RemoveGlobalSkillData, error) {
	return h.client.RemoveGlobalSkill(ctx, skillID, agents, force)
}

func (h *hubBridgeClient) PreviewFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
	return h.client.PreviewFeedback(ctx, req)
}

func (h *hubBridgeClient) ApplyFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
	return h.client.ApplyFeedback(ctx, req)
}

func serviceBridgeEnabled() bool {
	value := strings.TrimSpace(os.Getenv("SKILL_HUB_DISABLE_SERVICE_BRIDGE"))
	return value == "" || value == "0" || strings.EqualFold(value, "false")
}

func serviceBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("SKILL_HUB_SERVICE_URL")); value != "" {
		return value
	}
	return defaultServiceURL
}

func serviceSecretKey() string {
	return strings.TrimSpace(os.Getenv("SKILL_HUB_SERVICE_SECRET_KEY"))
}

// Resolve returns a remote client only after a bounded health check succeeds.
// Callers must use their local runtime when Resolve returns false.
func Resolve() (Client, bool) {
	if !serviceBridgeEnabled() {
		return nil, false
	}

	client := hubclientmodule.NewWithSecret(serviceBaseURL(), serviceSecretKey())
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	bridgeClient := &hubBridgeClient{client: client}
	if !bridgeClient.Available(ctx) {
		return nil, false
	}
	return bridgeClient, true
}
