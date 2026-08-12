package phpfpm

import (
	"fmt"
	"os"
	"syscall"
)

// Autodiscovery finds PHP-FPM masters by scanning the process table and then
// executes the binary it found, with a config path taken from that process's
// command line. Running as root -- which autodiscovery effectively requires,
// since reading another user's /proc/<pid>/exe needs CAP_SYS_PTRACE -- that is
// a path from "any local user can start a process" to "the exporter runs their
// code as root".
//
// So a discovered binary and its config file are used only if they are owned by
// root or by us, and are not writable by anyone else. Ownership is the check
// that matters: a file only root can replace can only have been placed there by
// root.

// trustedPath reports whether a discovered path is safe to execute or read.
func trustedPath(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("cannot stat %s: %w", path, err)
	}

	// A symlink can be repointed after the check, and its own ownership says
	// nothing about the target's.
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", path)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		// Unknown platform: refuse rather than guess.
		return fmt.Errorf("cannot determine ownership of %s", path)
	}

	if uid := os.Getuid(); stat.Uid != 0 && uint64(stat.Uid) != uint64(uid) {
		return fmt.Errorf("%s is owned by uid %d, expected root or uid %d", path, stat.Uid, uid)
	}

	// Group- or world-writable means someone other than the owner can swap the
	// contents out from under us between this check and the exec.
	if perm := info.Mode().Perm(); perm&0o022 != 0 {
		return fmt.Errorf("%s is writable by group or others (mode %04o)", path, perm)
	}

	return nil
}
