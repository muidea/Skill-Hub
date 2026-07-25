package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var skillHeadingPattern = regexp.MustCompile(`^#{1,2}\s+(.+?)\s*$`)
var numberedHeadingPrefix = regexp.MustCompile(`^\d+[.)、]\s*`)

// deprecatedFrontmatterRoots contains historical metadata that no longer has
// a consumer. Its removal must not make an otherwise complete skill update
// fail the archive integrity check.
var deprecatedFrontmatterRoots = map[string]struct{}{
	"compatibility": {},
}

// EnsureCoreInformationPreserved rejects a skill update that would discard
// information already present in the archived copy. It deliberately focuses on
// stable metadata, top-level guidance sections, and reusable skill resources.
func EnsureCoreInformationPreserved(existingDir, candidateDir string) error {
	existingContent, err := os.ReadFile(filepath.Join(existingDir, "SKILL.md"))
	if err != nil {
		return fmt.Errorf("读取现有 SKILL.md 失败: %w", err)
	}
	candidateContent, err := os.ReadFile(filepath.Join(candidateDir, "SKILL.md"))
	if err != nil {
		return fmt.Errorf("读取待更新 SKILL.md 失败: %w", err)
	}
	existingFrontmatter, err := ParseFrontmatter(existingContent)
	if err != nil {
		return fmt.Errorf("现有 SKILL.md 无效，拒绝覆盖: %w", err)
	}
	candidateFrontmatter, err := ParseFrontmatter(candidateContent)
	if err != nil {
		return fmt.Errorf("待更新 SKILL.md 无效: %w", err)
	}

	var omissions []string
	for _, field := range missingCoreFrontmatterFields(existingFrontmatter, candidateFrontmatter) {
		omissions = append(omissions, "frontmatter."+field)
	}
	if compareSkillVersions(ExtractVersion(candidateContent), ExtractVersion(existingContent)) < 0 {
		omissions = append(omissions, fmt.Sprintf("version (%s -> %s)", ExtractVersion(existingContent), ExtractVersion(candidateContent)))
	}
	for _, heading := range missingHeadings(existingContent, candidateContent) {
		omissions = append(omissions, "section: "+heading)
	}
	for _, path := range missingCoreFiles(existingDir, candidateDir) {
		omissions = append(omissions, "file: "+path)
	}
	if len(omissions) == 0 {
		return nil
	}
	sort.Strings(omissions)
	return fmt.Errorf("更新会丢失既有技能核心信息: %s", strings.Join(omissions, ", "))
}

func missingCoreFrontmatterFields(existing, candidate map[string]interface{}) []string {
	existingFields := flattenFrontmatter(existing, "")
	candidateFields := flattenFrontmatter(candidate, "")
	var missing []string
	for field := range existingFields {
		if field == "version" || field == "metadata.version" {
			continue
		}
		if isDeprecatedFrontmatterField(field) {
			continue
		}
		if _, ok := candidateFields[field]; !ok {
			missing = append(missing, field)
		}
	}
	if hasExplicitVersion(existing) && !hasExplicitVersion(candidate) {
		missing = append(missing, "version")
	}
	return missing
}

func isDeprecatedFrontmatterField(field string) bool {
	root, _, _ := strings.Cut(field, ".")
	_, ok := deprecatedFrontmatterRoots[root]
	return ok
}

func hasExplicitVersion(frontmatter map[string]interface{}) bool {
	if _, ok := frontmatter["version"]; ok {
		return true
	}
	metadata, ok := frontmatter["metadata"].(map[string]interface{})
	if !ok {
		return false
	}
	_, ok = metadata["version"]
	return ok
}

func flattenFrontmatter(value interface{}, prefix string) map[string]struct{} {
	fields := make(map[string]struct{})
	var visit func(interface{}, string)
	visit = func(current interface{}, path string) {
		switch typed := current.(type) {
		case map[string]interface{}:
			for key, nested := range typed {
				nestedPath := key
				if path != "" {
					nestedPath = path + "." + key
				}
				visit(nested, nestedPath)
			}
		default:
			if path != "" {
				fields[path] = struct{}{}
			}
		}
	}
	visit(value, prefix)
	return fields
}

func missingHeadings(existingContent, candidateContent []byte) []string {
	existing := skillHeadings(existingContent)
	candidate := skillHeadings(candidateContent)
	var missing []string
	for heading := range existing {
		if _, ok := candidate[heading]; !ok {
			missing = append(missing, heading)
		}
	}
	return missing
}

func skillHeadings(content []byte) map[string]struct{} {
	headings := make(map[string]struct{})
	for _, line := range strings.Split(string(content), "\n") {
		matches := skillHeadingPattern.FindStringSubmatch(line)
		if len(matches) != 2 {
			continue
		}
		heading := strings.ToLower(strings.TrimSpace(numberedHeadingPrefix.ReplaceAllString(matches[1], "")))
		if heading != "" {
			headings[heading] = struct{}{}
		}
	}
	return headings
}

func missingCoreFiles(existingDir, candidateDir string) []string {
	existing, err := coreSkillFiles(existingDir)
	if err != nil {
		return []string{"<无法读取现有资源>"}
	}
	candidate, err := coreSkillFiles(candidateDir)
	if err != nil {
		return []string{"<无法读取待更新资源>"}
	}
	var missing []string
	for path := range existing {
		if _, ok := candidate[path]; !ok {
			missing = append(missing, path)
		}
	}
	return missing
}

func coreSkillFiles(skillDir string) (map[string]struct{}, error) {
	files := make(map[string]struct{})
	err := filepath.WalkDir(skillDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == skillDir {
			return nil
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > 1 && (parts[0] == "references" || parts[0] == "scripts" || parts[0] == "assets" || parts[0] == "agents") {
			files[filepath.ToSlash(rel)] = struct{}{}
		}
		return nil
	})
	return files, err
}

func compareSkillVersions(left, right string) int {
	leftParts := skillVersionParts(left)
	rightParts := skillVersionParts(right)
	for i := 0; i < len(leftParts) || i < len(rightParts); i++ {
		var leftPart, rightPart int
		if i < len(leftParts) {
			leftPart = leftParts[i]
		}
		if i < len(rightParts) {
			rightPart = rightParts[i]
		}
		if leftPart > rightPart {
			return 1
		}
		if leftPart < rightPart {
			return -1
		}
	}
	return 0
}

func skillVersionParts(version string) []int {
	version = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V"))
	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		digits := ""
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			digits += string(r)
		}
		value, _ := strconv.Atoi(digits)
		result = append(result, value)
	}
	return result
}
