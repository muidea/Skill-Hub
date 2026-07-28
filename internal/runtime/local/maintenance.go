package local

import adapterpkg "github.com/muidea/skill-hub/internal/adapter"

// Maintenance contains filesystem maintenance operations that do not own
// repository, project or global state.
type Maintenance struct{ cleanup func(string) error }

func NewMaintenance() *Maintenance {
	return &Maintenance{cleanup: adapterpkg.CleanupTimestampedBackupDirs}
}

func (m *Maintenance) CleanupTimestampedBackupDirs(basePath string) error {
	return m.cleanup(basePath)
}
