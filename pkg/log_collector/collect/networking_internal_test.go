package collect

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkdUnitFiles(t *testing.T) {
	root := t.TempDir()
	// units across two config roots, plus a drop-in directory that must be
	// listed as a unit path, not descended into
	require.NoError(t, os.MkdirAll(filepath.Join(root, "usr/lib/systemd/network"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "etc/systemd/network/10-eth0.network.d"), 0o755))
	for _, f := range []string{
		"usr/lib/systemd/network/99-default.link",
		"etc/systemd/network/10-eth0.network",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(root, f), []byte("[Match]\n"), 0o644))
	}

	acc, err := NewAccessor(context.Background(), Config{Root: root, Destination: t.TempDir()})
	require.NoError(t, err)

	units, err := networkdUnitFiles(acc)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/usr/lib/systemd/network/99-default.link",
		"/etc/systemd/network/10-eth0.network",
	}, units)
}

func TestNetworkdUnitFilesNoConfig(t *testing.T) {
	acc, err := NewAccessor(context.Background(), Config{Root: t.TempDir(), Destination: t.TempDir()})
	require.NoError(t, err)

	units, err := networkdUnitFiles(acc)
	require.NoError(t, err)
	assert.Empty(t, units)
}

func TestDumpNetworkdUnits(t *testing.T) {
	bin := t.TempDir()
	// echo the unit argument so the output file records which unit was dumped
	script := `#!/bin/sh
echo "config of $2"
`
	require.NoError(t, os.WriteFile(filepath.Join(bin, "systemd-analyze"), []byte(script), 0o755))
	t.Setenv("PATH", bin)

	dst := t.TempDir()
	// Root must be "/" here: a non-root Root makes osext chroot the command,
	// which an unprivileged test cannot do.
	acc, err := NewAccessor(context.Background(), Config{Root: "/", Destination: dst})
	require.NoError(t, err)

	units := []string{"/usr/lib/systemd/network/99-default.link", "/etc/systemd/network/10-eth0.network"}
	require.NoError(t, dumpNetworkdUnits(acc, units))

	for _, unit := range units {
		got, err := os.ReadFile(filepath.Join(dst, "networking/systemd-network", filepath.Base(unit)))
		require.NoError(t, err, unit)
		assert.Equal(t, "config of "+unit+"\n", string(got), unit)
	}
}
