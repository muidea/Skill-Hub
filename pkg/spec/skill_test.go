package spec

import "testing"

func TestNormalizeSkillStatus(t *testing.T) {
	tests := map[string]string{
		"Synced":                      StatusSynced,
		"ok":                          StatusSynced,
		"Modified":                    StatusModified,
		"Outdated":                    StatusOutdated,
		"stale":                       StatusOutdated,
		"ModifiedAgainstOutdatedRepo": StatusDiverged,
		"Missing":                     StatusMissing,
		"not_applied":                 StatusMissing,
		"missing_agent_dir":           StatusUnavailable,
		"error":                       StatusUnavailable,
	}
	for input, want := range tests {
		if got := NormalizeSkillStatus(input); got != want {
			t.Errorf("NormalizeSkillStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLegacyProjectSkillStatus(t *testing.T) {
	if got := LegacyProjectSkillStatus(StatusDiverged); got != "ModifiedAgainstOutdatedRepo" {
		t.Errorf("LegacyProjectSkillStatus(diverged) = %q", got)
	}
}
