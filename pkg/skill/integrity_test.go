package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCoreInformationPreservedAllowsRetiredCompatibilityRemoval(t *testing.T) {
	existingDir := writeIntegritySkill(t, "---\nname: test\ndescription: test skill\ncompatibility: legacy\nmetadata:\n  version: 1.0.0\n---\n# Test\n\n## Workflow\n")
	candidateDir := writeIntegritySkill(t, "---\nname: test\ndescription: test skill\nmetadata:\n  version: 1.0.1\n---\n# Test\n\n## Workflow\n")

	if err := EnsureCoreInformationPreserved(existingDir, candidateDir); err != nil {
		t.Fatalf("EnsureCoreInformationPreserved() error = %v, want nil", err)
	}
}

func TestEnsureCoreInformationPreservedRejectsRequiredInformationRemoval(t *testing.T) {
	existingDir := writeIntegritySkill(t, "---\nname: test\ndescription: test skill\nmetadata:\n  version: 1.0.0\n---\n# Test\n\n## Workflow\n")
	candidateDir := writeIntegritySkill(t, "---\nname: test\nmetadata:\n  version: 1.0.1\n---\n# Test\n")

	err := EnsureCoreInformationPreserved(existingDir, candidateDir)
	if err == nil {
		t.Fatal("EnsureCoreInformationPreserved() error = nil, want required information removal error")
	}
	for _, omission := range []string{"frontmatter.description", "section: workflow"} {
		if !strings.Contains(err.Error(), omission) {
			t.Errorf("error = %q, want omission %q", err, omission)
		}
	}
}

func writeIntegritySkill(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return dir
}
