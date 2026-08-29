package fleetops

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeCert(t *testing.T) (path string, der []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "pve1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	path = filepath.Join(t.TempDir(), "pve-ssl.pem")
	if err := os.WriteFile(path,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path, der
}

// The fingerprint must be over the DER. Hashing the PEM file would change with
// whitespace and match nothing Proxmox prints — a value that looks right,
// compares unequal, and sends whoever debugs it to the wrong place.
func TestFingerprintIsOverTheCertificateNotTheFile(t *testing.T) {
	path, der := writeCert(t)
	old := LocalCertPath
	LocalCertPath = path
	t.Cleanup(func() { LocalCertPath = old })

	got, err := LocalFingerprint()
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	sum := sha256.Sum256(der)
	parts := make([]string, 0, len(sum))
	for _, b := range sum {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	if want := strings.Join(parts, ":"); got != want {
		t.Fatalf("fingerprint = %s, want %s", got, want)
	}
	if strings.ToLower(got) == got {
		t.Fatal("fingerprint is lowercase; Proxmox prints uppercase colon-separated hex")
	}
}

func TestFingerprintRefusesSomethingThatIsNotACertificate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "junk.pem")
	if err := os.WriteFile(path,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("nope")}), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	old := LocalCertPath
	LocalCertPath = path
	t.Cleanup(func() { LocalCertPath = old })

	if _, err := LocalFingerprint(); err == nil {
		t.Fatal("a PEM private key was accepted as a certificate")
	}
}

// Host keys are reported in the two-field form known_hosts expects, with the
// trailing comment dropped.
//
// "<type> <base64> root@pve3" carries an identity ssh does not read and that
// changes when a hostname does — a pin that includes it still works, but the
// stored value then differs from the same key on a renamed host, which reads as
// a key change to anyone comparing them.
func TestLocalHostKeysDropTheComment(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ssh_host_ed25519_key.pub")
	if err := os.WriteFile(p, []byte("ssh-ed25519 AAAAC3NzaC1lZDI1 root@pve3\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	old := HostKeyPaths
	HostKeyPaths = []string{p, filepath.Join(dir, "absent.pub")}
	t.Cleanup(func() { HostKeyPaths = old })

	got, err := LocalHostKeys()
	if err != nil {
		t.Fatalf("host keys: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("keys = %v, want 1 (a missing key type is not an error)", got)
	}
	if got[0] != "ssh-ed25519 AAAAC3NzaC1lZDI1" {
		t.Fatalf("key = %q, want the comment dropped", got[0])
	}
}

// A host with no readable host keys says so, rather than reporting an empty set
// that would blank what peers pin to reach it.
func TestLocalHostKeysRefusesWhenThereAreNone(t *testing.T) {
	old := HostKeyPaths
	HostKeyPaths = []string{filepath.Join(t.TempDir(), "absent.pub")}
	t.Cleanup(func() { HostKeyPaths = old })

	if _, err := LocalHostKeys(); err == nil {
		t.Fatal("a host with no keys reported success")
	}
}
