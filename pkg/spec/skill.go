package spec

// Skill 表示一个技能的完整定义
type Skill struct {
	ID            string        `yaml:"id" json:"id"`
	Name          string        `yaml:"name" json:"name"`
	Version       string        `yaml:"version" json:"version"`
	Author        string        `yaml:"author" json:"author"`
	Description   string        `yaml:"description" json:"description"`
	Tags          []string      `yaml:"tags" json:"tags"`
	Compatibility Compatibility `yaml:"compatibility" json:"compatibility"`
	Variables     []Variable    `yaml:"variables" json:"variables"`
	Dependencies  []string      `yaml:"dependencies" json:"dependencies"`
	Claude        *ClaudeConfig `yaml:"claude,omitempty" json:"claude,omitempty"`
}

// ClaudeConfig Claude专项配置
type ClaudeConfig struct {
	Mode       string    `yaml:"mode,omitempty" json:"mode,omitempty"` // instruction | tool
	Runtime    string    `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Entrypoint string    `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	ToolSpec   *ToolSpec `yaml:"tool_spec,omitempty" json:"tool_spec,omitempty"`
}

// ToolSpec 工具定义规范
type ToolSpec struct {
	Name        string                 `yaml:"name" json:"name"`
	Description string                 `yaml:"description" json:"description"`
	InputSchema map[string]interface{} `yaml:"input_schema" json:"input_schema"`
}

// Compatibility 表示技能支持的AI工具
type Compatibility struct {
	Cursor     bool `yaml:"cursor" json:"cursor"`
	ClaudeCode bool `yaml:"claude_code" json:"claude_code"`
	Shell      bool `yaml:"shell" json:"shell"`
}

// Variable 表示技能模板中的变量
type Variable struct {
	Name        string `yaml:"name" json:"name"`
	Default     string `yaml:"default" json:"default"`
	Description string `yaml:"description" json:"description"`
}

// SkillMetadata 用于技能索引的简化信息
type SkillMetadata struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Version       string        `json:"version"`
	Author        string        `json:"author"`
	Description   string        `json:"description"`
	Tags          []string      `json:"tags"`
	Compatibility Compatibility `json:"compatibility"`
}

// Registry 表示技能仓库的索引
type Registry struct {
	Version string          `json:"version"`
	Skills  []SkillMetadata `json:"skills"`
}

// ProjectState 表示项目与技能的关联状态
type ProjectState struct {
	ProjectPath string               `json:"project_path"`
	Skills      map[string]SkillVars `json:"skills"`
}

// SkillVars 表示项目中某个技能的变量配置
type SkillVars struct {
	SkillID   string            `json:"skill_id"`
	Version   string            `json:"version"`
	Variables map[string]string `json:"variables"`
}
