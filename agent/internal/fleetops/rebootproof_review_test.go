package fleetops

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// A host held on an older kernel through GRUB_DEFAULT — not a
// proxmox-boot-tool pin — used to be expected to boot the newest image, so
// under if-needed it was rebooted for a kernel it would never boot, failed
// with ErrBootedOldKernel, and did the same again every window.
func TestAGrubHeldKernelIsNotRebootedForAndNotFailed(t *testing.T) {
	for name, grub := range map[string]GrubSelection{
		"default names the kernel": {Default: "Advanced options for Proxmox VE GNU/Linux>Proxmox VE GNU/Linux, with Linux " + oldKernel},
		"saved entry by id":        {Default: "saved", Env: map[string]string{"saved_entry": "gnulinux-advanced-x>gnulinux-" + oldKernel + "-advanced-x"}},
		"a non-zero index":         {Default: "1>2"},
		"an unresolvable title":    {Default: "saved", Env: map[string]string{"saved_entry": "Memory test"}},
		"an evaluated expression":  {Default: "${KERNEL}"},
	} {
		t.Run(name, func(t *testing.T) {
			h := newProofHost(t)
			h.facts.grub = grub
			if err := h.run(`{"reboot":"if-needed","allow_guest_downtime":true}`); err != nil {
				t.Fatalf("err = %v", err)
			}
			if h.reboots != 0 {
				t.Fatalf("rebooted for a kernel GRUB will not boot (%d)", h.reboots)
			}
		})
	}
}

// Under reboot: always the held host still reboots — and comes back on the
// kernel GRUB chose, which must be accepted, not failed.
func TestAGrubHeldKernelIsAcceptedAfterAReboot(t *testing.T) {
	for name, grub := range map[string]GrubSelection{
		"resolved":   {Default: "saved", Env: map[string]string{"saved_entry": "Proxmox VE GNU/Linux, with Linux " + oldKernel}},
		"unresolved": {Default: "2"},
	} {
		t.Run(name, func(t *testing.T) {
			h := newProofHost(t)
			h.facts.grub = grub
			_ = h.run(specAlways)
			h.bootInto(oldKernel)
			if err := h.run(specAlways); err != nil {
				t.Fatalf("booting the kernel GRUB holds was failed: %v", err)
			}
		})
	}
}

// A resolved GRUB hold is still checked: booting something else is a failure.
func TestAResolvedGrubHoldIsStillChecked(t *testing.T) {
	h := newProofHost(t)
	h.facts.grub = GrubSelection{Default: "saved", Env: map[string]string{"saved_entry": "Proxmox VE GNU/Linux, with Linux " + oldKernel}}
	_ = h.run(specAlways)
	h.bootInto(newKernel)
	if err := h.run(specAlways); err == nil || !strings.Contains(err.Error(), "GRUB") {
		t.Fatalf("err = %v, want ErrBootedOldKernel naming GRUB", err)
	}
}

// grub-reboot's one-shot next_entry wins over everything else in GRUB.
func TestGrubNextEntryIsTheExpectation(t *testing.T) {
	got, src, err := expectedKernel(&fakeFacts{
		kernels: []string{oldKernel, newKernel},
		grub:    GrubSelection{Default: "0", Env: map[string]string{"next_entry": "gnulinux-" + oldKernel + "-advanced-x"}},
	})
	if err != nil || got != oldKernel || src == "" {
		t.Fatalf("got %q from %q (%v), want %s", got, src, err, oldKernel)
	}
}

// The usual case stays as it was: GRUB_DEFAULT unset, 0, or saved with
// nothing saved all boot the first entry, which update-grub makes the newest.
func TestGrubDefaultZeroIsTheNewestKernel(t *testing.T) {
	for _, g := range []GrubSelection{{}, {Default: "0"}, {Default: "saved"}} {
		got, _, err := expectedKernel(&fakeFacts{kernels: []string{oldKernel, newKernel}, grub: g})
		if err != nil || got != newKernel {
			t.Fatalf("%+v: got %q (%v), want %s", g, got, err, newKernel)
		}
	}
}

// -1 must not match -10.
func TestAGrubEntryMatchesTheWholeKernelRelease(t *testing.T) {
	for _, want := range []string{"6.8.12-1-pve", "6.8.12-10-pve"} {
		got, _, err := expectedKernel(&fakeFacts{
			kernels: []string{"6.8.12-1-pve", "6.8.12-10-pve"},
			grub:    GrubSelection{Default: "Proxmox VE GNU/Linux, with Linux " + want},
		})
		if err != nil || got != want {
			t.Fatalf("got %q (%v), want %s", got, err, want)
		}
	}
}

func TestParseGrubDefaults(t *testing.T) {
	sel := parseGrubDefault([]string{
		"# comment\nGRUB_DEFAULT=0\nGRUB_TIMEOUT=5\n",
		"GRUB_DEFAULT=\"saved\"\n",
	})
	if sel != "saved" {
		t.Fatalf("a grub.d override was not applied: %q", sel)
	}
	if got := parseGrubDefault([]string{"GRUB_DEFAULT='Proxmox VE, with Linux 6.8.12-4-pve'"}); got != "Proxmox VE, with Linux 6.8.12-4-pve" {
		t.Fatalf("single-quoted value: %q", got)
	}
	env := parseGrubEnv("# GRUB Environment Block\nsaved_entry=gnulinux-6.8.12-4-pve-advanced-x\n#########")
	if env["saved_entry"] != "gnulinux-6.8.12-4-pve-advanced-x" || len(env) != 1 {
		t.Fatalf("env = %v", env)
	}
}

// The changed-package list used to be logged whole, twice. A full PVE upgrade
// changes hundreds of packages: over 8 KiB the line was cut and no longer JSON,
// over 16 KiB the log batch was refused and resent forever.
func TestProofLinesStayBoundedForAHugeUpgrade(t *testing.T) {
	h := newProofHost(t)
	h.det.onRun = func(key string) {
		if strings.HasSuffix(key, "apt-dist-upgrade") {
			for i := range 2000 {
				h.facts.pkgs[fmt.Sprintf("lib-a-rather-long-package-name-%04d", i)] = "1.2.3-4+deb12u5"
			}
			h.facts.kernels = append(h.facts.kernels, newKernel)
		}
	}
	_ = h.run(specAlways)
	h.bootInto(newKernel)
	if err := h.run(specAlways); err != nil {
		t.Fatal(err)
	}
	var seen int
	for _, l := range h.logs {
		if !strings.HasPrefix(l, proofLogPrefix) {
			continue
		}
		seen++
		if len(l) > maxProofLineBytes {
			t.Fatalf("a %d-byte proof line; the bound is %d: %.200s…", len(l), maxProofLineBytes, l)
		}
		_, js, _ := strings.Cut(l, ": ")
		if !json.Valid([]byte(js)) {
			t.Fatalf("not valid JSON: %.200s…", l)
		}
	}
	if seen < 3 || !h.logged(`"changed_count":2000`) {
		t.Fatalf("the count of changed packages was not logged (%d lines): %v", seen, h.logs)
	}
	// The full list is not lost: it is in the record file.
	x, err := proofFor(h.deps, h.env).load()
	if err != nil || len(x.Changed) != 2000 {
		t.Fatalf("record holds %d changes (%v)", len(x.Changed), err)
	}
}

// Resumed in phaseVerifying — killed during the check after the reboot — must
// check again, never upgrade and reboot a second time.
func TestResumingInVerifyingDoesNotRebootAgain(t *testing.T) {
	h := newProofHost(t)
	_ = h.run(specAlways)
	h.bootInto(newKernel)
	h.phase = phaseVerifying
	if err := h.run(specAlways); err != nil {
		t.Fatal(err)
	}
	if h.reboots != 1 {
		t.Fatalf("rebooted %d times; resuming the check rebooted again", h.reboots)
	}
}

// pve-qemu-kvm is a reason, not a trigger: the guests need restarting onto
// it, not the host.
func TestPveQemuKvmAloneDoesNotReboot(t *testing.T) {
	h := newProofHost(t)
	h.facts.pkgs["pve-qemu-kvm"] = "9.0.2-1"
	h.det.onRun = func(key string) {
		if strings.HasSuffix(key, "apt-dist-upgrade") {
			h.facts.pkgs["pve-qemu-kvm"] = "9.0.2-3"
		}
	}
	if err := h.run(`{"reboot":"if-needed","allow_guest_downtime":true}`); err != nil {
		t.Fatal(err)
	}
	if h.reboots != 0 {
		t.Fatal("rebooted for pve-qemu-kvm alone")
	}
	if !h.logged("new pve-qemu-kvm 9.0.2-3") {
		t.Fatalf("pve-qemu-kvm was not recorded as a reason: %v", h.logs)
	}
}
