//go:build !windows

package state

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockStateFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}

func unlockStateFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
