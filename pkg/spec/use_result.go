package spec

// UseResult is the canonical result of enabling a skill. Both project and
// machine-global use operations return this shape so CLI and HTTP callers can
// consume the same identifiers, source, version and target scope.
type UseResult struct {
	Scope       string   `json:"scope"`
	ProjectPath string   `json:"project_path,omitempty"`
	SkillID     string   `json:"skill_id"`
	Version     string   `json:"version"`
	Repository  string   `json:"repository"`
	Agents      []string `json:"agents,omitempty"`
	Status      string   `json:"status"`
}
