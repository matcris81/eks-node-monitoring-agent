package collect

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeStub creates an executable shell script named name in bin.
func writeStub(t *testing.T, bin, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755))
}

func TestSociSnapshotter(t *testing.T) {
	t.Run("unit active collects status and journal", func(t *testing.T) {
		bin := t.TempDir()
		writeStub(t, bin, "systemctl", `case "$1" in
  is-active) exit 0 ;;
  status) echo "soci-snapshotter active (running)" ;;
esac`)
		writeStub(t, bin, "journalctl", `echo "soci journal"`)
		t.Setenv("PATH", bin)

		dst := t.TempDir()
		// Root must be "/" here: a non-root Root makes osext chroot the
		// command, which an unprivileged test cannot do. The config.toml copy
		// is covered separately with a fake root.
		acc, err := NewAccessor(context.Background(), Config{Root: "/", Destination: dst})
		require.NoError(t, err)

		require.NoError(t, sociSnapshotter(acc))

		status, err := os.ReadFile(filepath.Join(dst, "containerd/soci-snapshotter/soci-snapshotter-status.txt"))
		require.NoError(t, err)
		assert.Equal(t, "soci-snapshotter active (running)\n", string(status))

		journal, err := os.ReadFile(filepath.Join(dst, "containerd/soci-snapshotter/soci-snapshotter-log.txt"))
		require.NoError(t, err)
		assert.Equal(t, "soci journal\n", string(journal))
	})

	t.Run("unit not active skips without error", func(t *testing.T) {
		bin := t.TempDir()
		writeStub(t, bin, "systemctl", `exit 3`)
		t.Setenv("PATH", bin)

		dst := t.TempDir()
		acc, err := NewAccessor(context.Background(), Config{Root: "/", Destination: dst})
		require.NoError(t, err)

		require.NoError(t, sociSnapshotter(acc))
		_, err = os.Stat(filepath.Join(dst, "containerd/soci-snapshotter"))
		assert.True(t, os.IsNotExist(err), "no output expected when the unit is not active")
	})
}
