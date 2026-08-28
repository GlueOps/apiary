package fleetops

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// selfSignedCAPEM mints a real CA certificate. AppendCertsFromPEM rejects
// anything that is not one, so a placeholder would silently exercise the
// fallback path instead of the case under test.
func selfSignedCAPEM(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Proxmox Virtual Environment"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// THE LOCAL API IS VERIFIED AGAINST THE NODE'S OWN CA.
//
// The agent talks to https://127.0.0.1:8006, whose certificate is issued by a CA
// that exists only on that node. Verifying against the system trust store — which
// does not contain it — failed every task that reads cluster state, including the
// dist-upgrade's post-reboot check:
//
//	tls: failed to verify certificate: x509: certificate signed by unknown authority
func TestLocalAPIClientTrustsTheNodesOwnCA(t *testing.T) {
	dir := t.TempDir()
	ca := filepath.Join(dir, "pve-root-ca.pem")

	saved := pveRootCAPath
	pveRootCAPath = ca
	t.Cleanup(func() { pveRootCAPath = saved })

	// A real, self-signed CA certificate: AppendCertsFromPEM rejects anything
	// that is not one, so a placeholder would test the fallback instead.
	if err := os.WriteFile(ca, selfSignedCAPEM(t), 0o644); err != nil {
		t.Fatal(err)
	}

	c := localAPIClient()
	tr, ok := c.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil {
		t.Fatal("no TLS configuration on the local API client")
	}
	if tr.TLSClientConfig.RootCAs == nil {
		t.Fatal("the node's own CA was not trusted, so the local API cannot be verified")
	}
	if tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("verification was disabled; a missing or unreadable CA is not a reason to " +
			"trust whatever answers on the port")
	}
}

// A CA that cannot be read falls back to the system roots. It does NOT disable
// verification: the connection failing is a better outcome than trusting
// anything that answers.
func TestAnUnreadableCADoesNotDisableVerification(t *testing.T) {
	saved := pveRootCAPath
	pveRootCAPath = filepath.Join(t.TempDir(), "absent.pem")
	t.Cleanup(func() { pveRootCAPath = saved })

	c := localAPIClient()
	tr := c.Transport.(*http.Transport)
	if tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("a missing CA disabled certificate verification")
	}
	if tr.TLSClientConfig.RootCAs != nil {
		t.Fatal("a CA was trusted from a file that does not exist")
	}
}
