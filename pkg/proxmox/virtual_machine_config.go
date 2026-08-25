package proxmox

func (vmc *VirtualMachineConfig) MergeDisks() map[string]string {
	merged := make(map[string]string, len(vmc.IDEs)+len(vmc.SCSIs)+len(vmc.SATAs)+len(vmc.VirtIOs))
	for _, m := range []map[string]string{vmc.IDEs, vmc.SCSIs, vmc.SATAs, vmc.VirtIOs} {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}
