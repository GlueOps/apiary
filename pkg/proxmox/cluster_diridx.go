package proxmox

import (
	"context"
)

func (cl *Cluster) Subdirs(ctx context.Context) ([]string, error) {
	return cl.clusterDiridx(ctx, "/cluster")
}

func (cl *Cluster) ACMESubdirs(ctx context.Context) ([]string, error) {
	return cl.acmeDiridx(ctx, "/cluster/acme")
}

func (cl *Cluster) FirewallSubdirs(ctx context.Context) ([]string, error) {
	return cl.firewallDiridx(ctx, "/cluster/firewall")
}

func (cl *Cluster) SDNSubdirs(ctx context.Context) ([]string, error) {
	return cl.sdnDiridx(ctx, "/cluster/sdn")
}

func (cl *Cluster) CephSubdirs(ctx context.Context) ([]string, error) {
	return cl.cephDiridx(ctx, "/cluster/ceph")
}

func (cl *Cluster) ConfigSubdirs(ctx context.Context) ([]string, error) {
	return cl.configDiridx(ctx, "/cluster/config")
}

func (cl *Cluster) HASubdirs(ctx context.Context) ([]string, error) {
	return cl.haDiridx(ctx, "/cluster/ha")
}

func (cl *Cluster) HAStatusSubdirs(ctx context.Context) ([]string, error) {
	return cl.haDiridx(ctx, "/cluster/ha/status")
}

func (cl *Cluster) QEMUSubdirs(ctx context.Context) ([]string, error) {
	return cl.qemuDiridx(ctx, "/cluster/qemu")
}

func (cl *Cluster) clusterDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) acmeDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) firewallDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) sdnDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) cephDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) configDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) haDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) qemuDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) bulkActionDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) backupInfoDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) metricsDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}

func (cl *Cluster) notificationsDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, cl.client, path)
}
