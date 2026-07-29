package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/fs"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/muidea/skill-hub/pkg/utils"
)

// StateManager 管理项目状态
type StateManager struct {
	statePath string
	fs        fs.FileSystem
}

// GetStatePath 获取状态文件路径
func (m *StateManager) GetStatePath() string {
	return m.statePath
}

// StateFile 表示状态文件的完整结构
type StateFile struct {
	Projects map[string]spec.ProjectConfig `json:"projects"`
}

// NewStateManager 创建新的状态管理器
func NewStateManager() (*StateManager, error) {
	return NewStateManagerWithFS(&fs.RealFileSystem{})
}

// NewStateManagerWithFS 使用指定的文件系统创建状态管理器
func NewStateManagerWithFS(fileSystem fs.FileSystem) (*StateManager, error) {
	statePath, err := config.GetStatePath()
	if err != nil {
		return nil, err
	}

	return &StateManager{
		statePath: statePath,
		fs:        fileSystem,
	}, nil
}

// LoadProjectState 加载指定项目的状态
func (m *StateManager) LoadProjectState(projectPath string) (*spec.ProjectState, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 读取状态文件
	data, err := m.fs.ReadFile(m.statePath)
	if err != nil {
		if m.fs.IsNotExist(err) {
			// 文件不存在，返回空状态。preferred_target 仅保留历史兼容，不再设置默认值。
			return &spec.ProjectState{
				ProjectPath: absPath,
				Skills:      make(map[string]spec.SkillVars),
			}, nil
		}
		return nil, fmt.Errorf("读取状态文件失败: %w", err)
	}

	// 解析所有项目状态
	var allStates map[string]spec.ProjectState
	if err := json.Unmarshal(data, &allStates); err != nil {
		return nil, invalidStateFileError(m.statePath, err)
	}

	// 查找当前项目状态
	if state, exists := allStates[absPath]; exists {
		return &state, nil
	}

	// 项目状态不存在，创建新状态。preferred_target 仅保留历史兼容，不再设置默认值。
	return &spec.ProjectState{
		ProjectPath: absPath,
		Skills:      make(map[string]spec.SkillVars),
	}, nil
}

// LoadAllProjectStates 加载所有项目状态
func (m *StateManager) LoadAllProjectStates() (map[string]spec.ProjectState, error) {
	data, err := m.fs.ReadFile(m.statePath)
	if err != nil {
		if m.fs.IsNotExist(err) {
			return map[string]spec.ProjectState{}, nil
		}
		return nil, fmt.Errorf("读取状态文件失败: %w", err)
	}

	var allStates map[string]spec.ProjectState
	if err := json.Unmarshal(data, &allStates); err != nil {
		return nil, invalidStateFileError(m.statePath, err)
	}
	if allStates == nil {
		return map[string]spec.ProjectState{}, nil
	}
	return allStates, nil
}

// SaveProjectState 保存项目状态
func (m *StateManager) SaveProjectState(state *spec.ProjectState) error {
	if state == nil {
		return fmt.Errorf("保存项目状态失败: 项目状态不能为空")
	}
	projectPath, err := filepath.Abs(state.ProjectPath)
	if err != nil {
		return fmt.Errorf("保存项目状态失败: 获取项目绝对路径失败: %w", err)
	}
	normalizedState := *state
	normalizedState.ProjectPath = projectPath
	state = &normalizedState

	return m.withWriteLock(func() error {
		// 读取和写入必须在同一把跨进程锁内完成，否则两个 CLI/daemon
		// 进程可能互相覆盖，或在直接写入期间留下不完整的 JSON。
		allStates := make(map[string]spec.ProjectState)
		if data, err := m.fs.ReadFile(m.statePath); err == nil {
			if err := json.Unmarshal(data, &allStates); err != nil {
				return invalidStateFileError(m.statePath, err)
			}
		}

		allStates[projectPath] = *state
		return m.saveAllProjectStates(allStates)
	})
}

func (m *StateManager) SaveAllProjectStates(allStates map[string]spec.ProjectState) error {
	return m.withWriteLock(func() error {
		return m.saveAllProjectStates(allStates)
	})
}

func (m *StateManager) saveAllProjectStates(allStates map[string]spec.ProjectState) error {
	data, err := json.MarshalIndent(allStates, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}

	// 确保目录存在
	if err := m.fs.MkdirAll(filepath.Dir(m.statePath), 0755); err != nil {
		return utils.CreateDirErr(err, filepath.Dir(m.statePath))
	}

	if _, ok := m.fs.(*fs.RealFileSystem); ok {
		if err := writeFileAtomically(m.statePath, data, 0644); err != nil {
			return fmt.Errorf("原子写入状态文件失败: %w", err)
		}
		return nil
	}

	if err := m.fs.WriteFile(m.statePath, data, 0644); err != nil {
		return fmt.Errorf("写入状态文件失败: %w", err)
	}

	return nil
}

func (m *StateManager) withWriteLock(write func() error) error {
	// 仅真实文件系统需要跨进程锁；保留 mock 文件系统的可测试性。
	if _, ok := m.fs.(*fs.RealFileSystem); !ok {
		return write()
	}

	if err := os.MkdirAll(filepath.Dir(m.statePath), 0755); err != nil {
		return utils.CreateDirErr(err, filepath.Dir(m.statePath))
	}

	lockFile, err := os.OpenFile(m.statePath+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("打开状态文件锁失败: %w", err)
	}
	defer lockFile.Close()

	if err := lockStateFile(lockFile); err != nil {
		return fmt.Errorf("锁定状态文件失败: %w", err)
	}
	defer unlockStateFile(lockFile)

	return write()
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) (err error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err = temporary.Chmod(perm); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err = temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func invalidStateFileError(path string, err error) error {
	return fmt.Errorf("项目状态文件 %q 格式错误；请先备份并修复该 JSON 文件: %w", path, err)
}

func (m *StateManager) PruneInvalidProjectStates() ([]string, error) {
	allStates, err := m.LoadAllProjectStates()
	if err != nil {
		return nil, err
	}

	var removed []string
	updatedStates := make(map[string]spec.ProjectState, len(allStates))
	for projectPath, state := range allStates {
		if projectPath == "" {
			removed = append(removed, projectPath)
			continue
		}

		info, err := m.fs.Stat(projectPath)
		if err != nil {
			if m.fs.IsNotExist(err) {
				removed = append(removed, projectPath)
				continue
			}
			return nil, fmt.Errorf("检查项目路径失败: %w", err)
		}
		if !info.IsDir() {
			removed = append(removed, projectPath)
			continue
		}

		if state.ProjectPath != projectPath {
			state.ProjectPath = projectPath
		}
		updatedStates[projectPath] = state
	}

	if err := m.SaveAllProjectStates(updatedStates); err != nil {
		return nil, err
	}

	sort.Strings(removed)
	return removed, nil
}

// AddSkillToProject 添加技能到项目
func (m *StateManager) AddSkillToProject(projectPath, skillID, version string, variables map[string]string) error {
	return m.AddSkillToProjectWithSource(projectPath, skillID, version, "", variables)
}

// AddSkillToProjectWithSource 添加技能到项目并记录来源仓库。
func (m *StateManager) AddSkillToProjectWithSource(projectPath, skillID, version, sourceRepository string, variables map[string]string) error {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return err
	}
	if variables == nil {
		if existing, ok := state.Skills[skillID]; ok {
			variables = existing.Variables
		}
	}

	state.Skills[skillID] = spec.SkillVars{
		SkillID:          skillID,
		Version:          version,
		SourceRepository: sourceRepository,
		Variables:        variables,
	}

	return m.SaveProjectState(state)
}

// FindProjectByPath 通过路径查找项目（支持递归向上查找）
func (m *StateManager) FindProjectByPath(path string) (*spec.ProjectState, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 读取所有项目状态
	data, err := m.fs.ReadFile(m.statePath)
	if err != nil {
		if m.fs.IsNotExist(err) {
			return nil, nil // 文件不存在，返回nil
		}
		return nil, fmt.Errorf("读取状态文件失败: %w", err)
	}

	var allStates map[string]spec.ProjectState
	if err := json.Unmarshal(data, &allStates); err != nil {
		return nil, invalidStateFileError(m.statePath, err)
	}

	// 递归向上查找
	currentPath := absPath
	for {
		// 检查当前路径是否有绑定
		if state, exists := allStates[currentPath]; exists {
			if state.ProjectPath != currentPath {
				state.ProjectPath = currentPath
			}
			return &state, nil
		}

		// 到达根目录，停止查找
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			break
		}
		currentPath = parentPath
	}

	return nil, nil // 未找到
}

// RemoveSkillFromProject 从项目移除技能
func (m *StateManager) RemoveSkillFromProject(projectPath, skillID string) error {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return err
	}

	delete(state.Skills, skillID)
	return m.SaveProjectState(state)
}

// GetProjectSkills 获取项目的所有技能
func (m *StateManager) GetProjectSkills(projectPath string) (map[string]spec.SkillVars, error) {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return nil, err
	}
	return state.Skills, nil
}

// ProjectHasSkill 检查项目是否启用了指定技能
func (m *StateManager) ProjectHasSkill(projectPath, skillID string) (bool, error) {
	skills, err := m.GetProjectSkills(projectPath)
	if err != nil {
		return false, err
	}

	_, exists := skills[skillID]
	return exists, nil
}

// UpdateSkillVariables 更新项目中技能的变量值
func (m *StateManager) UpdateSkillVariables(projectPath, skillID string, variables map[string]string) error {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return err
	}

	// 检查技能是否存在
	skillVars, exists := state.Skills[skillID]
	if !exists {
		return fmt.Errorf("技能 '%s' 未在项目中启用", skillID)
	}

	// 更新变量值
	skillVars.Variables = variables
	state.Skills[skillID] = skillVars

	return m.SaveProjectState(state)
}
