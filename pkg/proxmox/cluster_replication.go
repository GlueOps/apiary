package proxmox

import (
	"context"
	"fmt"
)

func (cl *Cluster) ReplicationJobs(ctx context.Context) (jobs []*ReplicationJob, err error) {
	err = cl.client.Get(ctx, "/cluster/replication", &jobs)
	return
}

func (cl *Cluster) ReplicationJob(ctx context.Context, id string) (job *ReplicationJob, err error) {
	job = &ReplicationJob{}
	err = cl.client.Get(ctx, fmt.Sprintf("/cluster/replication/%s", id), job)
	return
}

func (cl *Cluster) NewReplicationJob(ctx context.Context, opts *ReplicationJobOptions) error {
	return cl.client.Post(ctx, "/cluster/replication", opts, nil)
}

func (cl *Cluster) ReplicationJobUpdate(ctx context.Context, id string, opts *ReplicationJobUpdateOption) error {
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/replication/%s", id), opts, nil)
}

func (cl *Cluster) ReplicationJobDelete(ctx context.Context, id string, force, keep bool) error {
	path := fmt.Sprintf("/cluster/replication/%s", id)
	query := ""
	if force {
		query = "force=1"
	}
	if keep {
		if query != "" {
			query += "&"
		}
		query += "keep=1"
	}
	if query != "" {
		path += "?" + query
	}
	return cl.client.Delete(ctx, path, nil)
}
