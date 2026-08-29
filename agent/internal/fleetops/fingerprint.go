package fleetops

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

// LocalCertPath is where Proxmox keeps this node's API certificate.
//
// A var so a test can point it elsewhere. Under /etc/pve, so it is the
// certificate this node is actually serving rather than one a caller supplied.
var LocalCertPath = "/etc/pve/local/pve-ssl.pem"

// LocalFingerprint is this host's own API certificate fingerprint, in the
// colon-separated uppercase form Proxmox uses.
//
// Computed from the DER, not from the PEM bytes. Fingerprinting the file would
// produce a digest that changes with whitespace and header formatting and
// matches nothing Proxmox ever prints — a value that looks right, compares
// unequal, and sends whoever debugs it looking at the wrong thing.
//
// Read fresh on each call rather than cached. A certificate is reissued by
// `pvecm updatecerts`, which a join and a departure both run, and a cached
// fingerprint would go stale at exactly the moment another node is about to
// join through this one.
func LocalFingerprint() (string, error) {
	raw, err := os.ReadFile(LocalCertPath)
	if err != nil {
		return "", fmt.Errorf("read this host's certificate: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("%s does not contain a PEM certificate", LocalCertPath)
	}
	// Parsed rather than hashed straight from block.Bytes, so a file that is PEM
	// but not a certificate is refused here instead of yielding a plausible
	// digest of the wrong thing.
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse this host's certificate: %w", err)
	}
	sum := sha256.Sum256(cert.Raw)
	parts := make([]string, 0, len(sum))
	for _, b := range sum {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	return strings.Join(parts, ":"), nil
}
