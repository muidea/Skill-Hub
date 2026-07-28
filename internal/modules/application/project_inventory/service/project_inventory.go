package service

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"

	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/pkg/errors"
)

type ProjectSummary struct {
	ID          string `json:"id"`
	ProjectPath string `json:"project_path"`
	SkillCount  int    `json:"skill_count"`
}

type ProjectDetail struct {
	ID          string `json:"id"`
	ProjectPath string `json:"project_path"`
	SkillCount  int    `json:"skill_count"`
}

type ProjectSkill struct {
	SkillID          string `json:"skill_id"`
	Version          string `json:"version,omitempty"`
	Status           string `json:"status,omitempty"`
	SourceRepository string `json:"source_repository,omitempty"`
}

type ProjectInventory struct {
	projectState projectstateport.ProjectState
}

func New(projectState projectstateport.ProjectState) *ProjectInventory {
	return &ProjectInventory{
		projectState: projectState,
	}
}

func (p *ProjectInventory) ListProjects() ([]ProjectSummary, error) {
	allStates, err := p.projectState.List()
	if err != nil {
		return nil, errors.Wrap(err, "ListProjects: 加载项目状态失败")
	}

	projects := make([]ProjectSummary, 0, len(allStates))
	for projectPath, state := range allStates {
		projects = append(projects, ProjectSummary{
			ID:          projectID(projectPath),
			ProjectPath: projectPath,
			SkillCount:  len(state.Skills),
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ProjectPath < projects[j].ProjectPath
	})
	return projects, nil
}

func (p *ProjectInventory) GetProject(id string) (*ProjectDetail, error) {
	allStates, err := p.projectState.List()
	if err != nil {
		return nil, errors.Wrap(err, "GetProject: 加载项目状态失败")
	}

	for projectPath, state := range allStates {
		if projectID(projectPath) != id {
			continue
		}
		return &ProjectDetail{
			ID:          id,
			ProjectPath: projectPath,
			SkillCount:  len(state.Skills),
		}, nil
	}

	return nil, errors.NewWithCode("GetProject", errors.ErrFileNotFound, "项目不存在")
}

func (p *ProjectInventory) ListProjectSkills(id string) ([]ProjectSkill, error) {
	allStates, err := p.projectState.List()
	if err != nil {
		return nil, errors.Wrap(err, "ListProjectSkills: 加载项目状态失败")
	}

	for projectPath, state := range allStates {
		if projectID(projectPath) != id {
			continue
		}

		skills := make([]ProjectSkill, 0, len(state.Skills))
		for skillID, skillVars := range state.Skills {
			skills = append(skills, ProjectSkill{
				SkillID:          skillID,
				Version:          skillVars.Version,
				Status:           skillVars.Status,
				SourceRepository: skillVars.SourceRepository,
			})
		}

		sort.Slice(skills, func(i, j int) bool {
			return skills[i].SkillID < skills[j].SkillID
		})
		return skills, nil
	}

	return nil, errors.NewWithCode("ListProjectSkills", errors.ErrFileNotFound, "项目不存在")
}

func projectID(projectPath string) string {
	sum := sha1.Sum([]byte(projectPath))
	return hex.EncodeToString(sum[:8])
}
