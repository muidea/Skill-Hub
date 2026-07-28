package git

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseBranchSyncStatus(t *testing.T) {
	tests := []struct {
		name                  string
		status                string
		wantAhead, wantBehind int
		wantDirty             bool
	}{
		{
			name:       "clean local ahead",
			status:     "# branch.oid deadbeef\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +2 -0\n",
			wantAhead:  2,
			wantBehind: 0,
		},
		{
			name:       "dirty and divergent",
			status:     "# branch.ab +1 -3\n1 .M N... 100644 100644 100644 abc abc README.md\n",
			wantAhead:  1,
			wantBehind: 3,
			wantDirty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ahead, behind, dirty := parseBranchSyncStatus(tt.status)
			if ahead != tt.wantAhead || behind != tt.wantBehind || dirty != tt.wantDirty {
				t.Fatalf("parseBranchSyncStatus() = (%d, %d, %t), want (%d, %d, %t)", ahead, behind, dirty, tt.wantAhead, tt.wantBehind, tt.wantDirty)
			}
		})
	}
}

func TestSyncReportsLocalAheadWithoutPulling(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	seedDir := filepath.Join(tmpDir, "seed")
	localDir := filepath.Join(tmpDir, "local")
	runGitCommand(t, tmpDir, "init", "--bare", remoteDir)
	runGitCommand(t, tmpDir, "init", "-b", "main", seedDir)
	runGitCommand(t, seedDir, "config", "user.name", "tester")
	runGitCommand(t, seedDir, "config", "user.email", "tester@example.com")
	runGitCommand(t, seedDir, "remote", "add", "origin", remoteDir)
	writeSystemGitCommit(t, seedDir, "README.md", "remote\n", "initial commit")
	runGitCommand(t, seedDir, "push", "origin", "main")
	runGitCommand(t, tmpDir, "clone", "--branch", "main", remoteDir, localDir)
	runGitCommand(t, localDir, "config", "user.name", "tester")
	runGitCommand(t, localDir, "config", "user.email", "tester@example.com")
	writeSystemGitCommit(t, localDir, "README.md", "local\n", "local commit")

	result, err := Sync(localDir)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.Status != "local_ahead" || result.Ahead != 1 || result.Behind != 0 {
		t.Fatalf("Sync() result = %+v, want local_ahead ahead=1 behind=0", result)
	}
	if got := runGitCommand(t, remoteDir, "show", "main:README.md"); got != "remote\n" {
		t.Fatalf("remote content = %q, Sync() must not publish local commits", got)
	}
}

func TestSyncReportsDivergentHistoryWithoutChangingWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	seedDir := filepath.Join(tmpDir, "seed")
	localDir := filepath.Join(tmpDir, "local")
	runGitCommand(t, tmpDir, "init", "--bare", remoteDir)
	runGitCommand(t, tmpDir, "init", "-b", "main", seedDir)
	runGitCommand(t, seedDir, "config", "user.name", "tester")
	runGitCommand(t, seedDir, "config", "user.email", "tester@example.com")
	runGitCommand(t, seedDir, "remote", "add", "origin", remoteDir)
	writeSystemGitCommit(t, seedDir, "README.md", "base\n", "initial commit")
	runGitCommand(t, seedDir, "push", "origin", "main")
	runGitCommand(t, tmpDir, "clone", "--branch", "main", remoteDir, localDir)
	runGitCommand(t, localDir, "config", "user.name", "tester")
	runGitCommand(t, localDir, "config", "user.email", "tester@example.com")
	writeSystemGitCommit(t, localDir, "README.md", "local\n", "local commit")
	writeSystemGitCommit(t, seedDir, "README.md", "remote\n", "remote commit")
	runGitCommand(t, seedDir, "push", "origin", "main")

	result, err := Sync(localDir)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.Status != "divergent" || result.Ahead != 1 || result.Behind != 1 {
		t.Fatalf("Sync() result = %+v, want divergent ahead=1 behind=1", result)
	}
	if got := runGitCommand(t, localDir, "show", "HEAD:README.md"); got != "local\n" {
		t.Fatalf("local content = %q, Sync() must not alter divergent worktree", got)
	}
}
