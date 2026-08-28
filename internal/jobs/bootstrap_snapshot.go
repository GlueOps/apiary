package jobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// BootstrapSnapshotFromDB reads everything the bootstrap gate and steps need, in
// one transaction, so the decision cannot be made against a half-changed
// picture — the same discipline as ArrivalSnapshotFromDB.
//
// Deliberately narrower than the arrival's. There is no target to reach, no
// peer's host keys to seed and no foreign cluster to be checked against:
// `pvecm create` acts on this host alone. What is left is the host itself, and
// the count that decides whether this is a bootstrap at all.
func BootstrapSnapshotFromDB(
	ctx context.Context, db *gorm.DB, hypervisorID, targetHiveID uuid.UUID,
) (BootstrapSnapshot, error) {
	var snap BootstrapSnapshot

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var host tenant.Hypervisor
		if err := tx.First(&host, "id = ?", hypervisorID).Error; err != nil {
			return fmt.Errorf("machine %s: %w", hypervisorID, err)
		}
		snap.CurrentHiveID = host.HiveID
		snap.HostAddress = host.Address
		snap.ProvisioningState = host.ProvisioningState

		if host.Name != nil {
			snap.HostNodeName = *host.Name
		}
		if snap.HostNodeName == "" {
			// The capacity sync matches its rows on (hive_id, name), so this is
			// what the first observation of the new cluster will have to find.
			// Derived from the address as the arrival does; a cluster created
			// here is read back moments later and a wrong guess is corrected
			// then, whereas an empty name is a row the sync creates a duplicate
			// beside.
			snap.HostNodeName = tenant.NodeNameFromAddress(host.Address)
		}

		if err := tx.First(&snap.TargetHive, "id = ?", targetHiveID).Error; err != nil {
			return fmt.Errorf("target hive %s: %w", targetHiveID, err)
		}

		var members int64
		if err := tx.Model(&tenant.Hypervisor{}).
			Where("hive_id = ?", targetHiveID).Count(&members).Error; err != nil {
			return fmt.Errorf("count target hive members: %w", err)
		}
		snap.HiveNodeCount = int(members)
		return nil
	})
	return snap, err
}
