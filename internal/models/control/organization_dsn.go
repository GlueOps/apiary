package control

import (
	"errors"
	"fmt"

	"github.com/glueops/apiary/internal/utils"
)

// ErrConnectionStringNotEncrypted marks an org whose connection_string was stored
// in plaintext, before DSNs were sealed (F6). Its IV and tag are empty and
// ConnectionString holds the DSN itself.
//
// A sentinel rather than a refusal, because callers differ on what it means:
// connecting a tenant must refuse it and ask for a re-provision, while
// destroying one still needs to know which database to drop.
var ErrConnectionStringNotEncrypted = errors.New("connection_string is not encrypted")

// DecryptConnectionString opens the org's sealed tenant DSN with the tenant-DSN
// key. The org id is the AAD, so a sealed DSN copied onto another org's row
// does not open.
//
// The one implementation of the rule, so every caller treats a legacy
// plaintext DSN the same way: ErrConnectionStringNotEncrypted, never a base64
// error from decoding a postgres:// URL.
func (o *Organization) DecryptConnectionString(dsnKey []byte) (string, error) {
	if o.ConnectionStringIV == "" || o.ConnectionStringTag == "" {
		return "", ErrConnectionStringNotEncrypted
	}
	ct, err1 := utils.DecodeB64(o.ConnectionString)
	iv, err2 := utils.DecodeB64(o.ConnectionStringIV)
	tag, err3 := utils.DecodeB64(o.ConnectionStringTag)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", fmt.Errorf("decode connection string components")
	}
	plain, err := utils.DecryptAESGCMWithAAD(ct, dsnKey, iv, tag, []byte(o.ID.String()))
	if err != nil {
		return "", fmt.Errorf("decrypt connection string: %w", err)
	}
	return string(plain), nil
}
