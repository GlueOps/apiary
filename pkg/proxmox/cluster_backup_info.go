package proxmox

import (
	"context"
	"errors"
	"fmt"
)

func (cl *Cluster) BackupInfoSubdirs(ctx context.Context) ([]string, error) {
	return cl.backupInfoDiridx(ctx, "/cluster/backup-info")
}

func (cl *Cluster) GuestsNotInBackup(ctx context.Context) (guests []*BackupGuestEntry, err error) {
	err = cl.client.Get(ctx, "/cluster/backup-info/not-backed-up", &guests)
	return
}

func (b *ClusterBackup) IncludedVolumes(ctx context.Context) (root *BackupIncludedVolumesRoot, err error) {
	if b.ID == "" {
		return nil, errors.New("backup id can not be empty")
	}
	root = &BackupIncludedVolumesRoot{}
	err = b.client.Get(ctx, fmt.Sprintf("/cluster/backup/%s/included_volumes", b.ID), root)
	return
}
