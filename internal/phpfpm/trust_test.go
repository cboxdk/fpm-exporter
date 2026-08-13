package phpfpm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrustedPath(t *testing.T) {
	dir := t.TempDir()

	safe := filepath.Join(dir, "safe")
	if err := os.WriteFile(safe, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := trustedPath(safe); err != nil {
		t.Errorf("Expected our own non-world-writable file to be trusted, got: %v", err)
	}

	// The shape of the attack: a file anyone can rewrite between the check and
	// the exec.
	writable := filepath.Join(dir, "world-writable")
	if err := os.WriteFile(writable, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	// Chmod explicitly: umask would strip the group/other write bit from the
	// mode passed to WriteFile, and then this case would not test anything.
	if err := os.Chmod(writable, 0777); err != nil {
		t.Fatal(err)
	}
	if err := trustedPath(writable); err == nil {
		t.Errorf("Expected a world-writable binary to be refused")
	}

	link := filepath.Join(dir, "link")
	if err := os.Symlink(safe, link); err != nil {
		t.Fatal(err)
	}
	if err := trustedPath(link); err == nil {
		t.Errorf("Expected a symlink to be refused")
	}

	if err := trustedPath(filepath.Join(dir, "missing")); err == nil {
		t.Errorf("Expected a missing path to be refused")
	}

	if err := trustedPath(dir); err == nil {
		t.Errorf("Expected a directory to be refused")
	}
}
