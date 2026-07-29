//go:build windows

package state

import "os"

// Windows' atomic rename protects readers from partial JSON. Keep this a
// no-op until the Windows-specific locking abstraction is introduced.
func lockStateFile(_ *os.File) error { return nil }

func unlockStateFile(_ *os.File) error { return nil }
