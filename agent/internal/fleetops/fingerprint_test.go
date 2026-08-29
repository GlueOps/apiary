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
