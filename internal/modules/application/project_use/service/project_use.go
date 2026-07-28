package service

import (
	"path/filepath"

	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

type UseResult = spec.UseResult

type ProjectUse struct {
	projectState projectstateport.ProjectState
	repository   repositoryport.ProjectSource
}

func New(projectState projectstateport.ProjectState, repository repositoryport.ProjectSource) *ProjectUse {
	return &ProjectUse{
		projectState: projectState,
		repository:   repository,
	}
}

func (p *ProjectUse) EnableSkill(projectPath, skillID, repoName string, variables map[string]string) (*UseResult, error) {
	if projectPath == "" {
		return nil, errors.NewWithCode("EnableSkill", errors.ErrInvalidInput, "项目路径不能为空")
	}
	if skillID == "" {
		return nil, errors.NewWithCode("EnableSkill", errors.ErrInvalidInput, "技能 ID 不能为空")
	}

	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 获取项目绝对路径失败")
	}

	candidates, err := p.repository.FindSkill(skillID)
	if err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 查找技能失败")
	}
	if len(candidates) == 0 {
		return nil, errors.SkillNotFound("EnableSkill", skillID)
	}

	selectedRepo := repoName
	if selectedRepo == "" {
		if len(candidates) != 1 {
			return nil, errors.NewWithCode("EnableSkill", errors.ErrInvalidInput, "技能存在多个候选仓库，必须指定 repository")
		}
		selectedRepo = candidates[0].Repository
	}

	fullSkill, err := p.repository.LoadSkill(skillID, selectedRepo)
	if err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 加载技能详情失败")
	}

	projectRoot := absProjectPath
	projectState, err := p.projectState.Find(absProjectPath)
	if err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 查找项目状态失败")
	}
	if projectState != nil && projectState.ProjectPath != "" {
		projectRoot = projectState.ProjectPath
	}

	if projectState == nil {
		projectState = &spec.ProjectState{ProjectPath: projectRoot, Skills: make(map[string]spec.SkillVars)}
	}
	if projectState.Skills == nil {
		projectState.Skills = make(map[string]spec.SkillVars)
	}
	projectState.Skills[skillID] = spec.SkillVars{SkillID: skillID, Version: fullSkill.Version, SourceRepository: selectedRepo, Variables: variables}
	if err := p.projectState.Save(*projectState); err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 保存项目状态失败")
	}

	return &UseResult{
		Scope:       "project",
		ProjectPath: projectRoot,
		SkillID:     skillID,
		Version:     fullSkill.Version,
		Repository:  selectedRepo,
		Status:      "enabled",
	}, nil
}
