package collect

import (
	"errors"
	"os"
	"path/filepath"
)

type Containerd struct{}

var _ Collector = (*CommonLogs)(nil)

func (c *Containerd) Collect(acc *Accessor) error {
	return errors.Join(
		ctrdlogs(acc),
		acc.CommandOutput([]string{"containerd", "config", "dump"}, "containerd/containerd-config.txt", CommandOptionsNone),
		acc.CommandOutput([]string{"journalctl", "-o", "short-iso-precise", "-u", "containerd"}, "containerd/containerd-log.txt", CommandOptionsNone),
		acc.CommandOutput([]string{"ctr", "version"}, "containerd/containerd-version.txt", CommandOptionsNone),
		acc.CommandOutput([]string{"ctr", "namespaces", "list"}, "containerd/containerd-namespaces.txt", CommandOptionsNone),
		acc.CommandOutput([]string{"ctr", "--namespace", "k8s.io", "images", "list"}, "containerd/containerd-images.txt", CommandOptionsNone),
		acc.CommandOutput([]string{"ctr", "--namespace", "k8s.io", "containers", "list"}, "containerd/containerd-containers.txt", CommandOptionsNone),
		acc.CommandOutput([]string{"ctr", "--namespace", "k8s.io", "tasks", "list"}, "containerd/containerd-tasks.txt", CommandOptionsNone),
		acc.CommandOutput([]string{"ctr", "--namespace", "k8s.io", "plugins", "list"}, "containerd/containerd-plugins.txt", CommandOptionsNone),
		sociSnapshotter(acc),
	)
}

// sociSnapshotter collects the SOCI snapshotter's unit status, journal and
// configuration when the soci-snapshotter unit is active. Image lazy-loading
// failures on SOCI-enabled nodes are undiagnosable without them. Nodes
// without the unit skip collection.
func sociSnapshotter(acc *Accessor) error {
	if _, err := acc.CombinedOutput("systemctl", "is-active", "--quiet", "soci-snapshotter"); err != nil {
		acc.logger.Info("soci-snapshotter unit not active, skipping SOCI log collection")
		return nil
	}
	var merr error
	merr = errors.Join(merr,
		acc.CommandOutput([]string{"systemctl", "status", "soci-snapshotter"}, "containerd/soci-snapshotter/soci-snapshotter-status.txt", CommandOptionsIgnoreFailure),
		acc.CommandOutput([]string{"journalctl", "-o", "short-iso-precise", "-u", "soci-snapshotter", "--since", "10 days ago"}, "containerd/soci-snapshotter/soci-snapshotter-log.txt", CommandOptionsNone),
	)
	configPath := filepath.Join(acc.cfg.Root, "/etc/soci-snapshotter-grpc/config.toml")
	if _, err := os.Stat(configPath); err == nil {
		merr = errors.Join(merr, acc.CopyFile(configPath, "containerd/soci-snapshotter/config.toml"))
	}
	return merr
}

func ctrdlogs(acc *Accessor) error {
	containerLogs, err := filepath.Glob(filepath.Join(acc.cfg.Root, "/tmp/containerd.*.stacks.log"))
	if err != nil {
		return err
	}
	var merr error
	for _, logPath := range containerLogs {
		merr = errors.Join(merr, acc.CopyFile(logPath, filepath.Join("containerd", filepath.Base(logPath))))
	}
	return merr
}
