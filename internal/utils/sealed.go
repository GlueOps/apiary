package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Sealed struct {
	KeyID uuid.UUID `gorm:"type:uuid" json:"-"`

	Ciphertext string `gorm:"type:text;not null;default:''" json:"-"`
	IV         string `gorm:"type:text;not null;default:''" json:"-"`
	Tag        string `gorm:"type:text;not null;default:''" json:"-"`
}

func AAD(table string, orgID, rowID uuid.UUID) []byte {
	var b strings.Builder
	b.Grow(len(table) + 74)
	b.WriteString(table)
	b.WriteByte(':')
	b.WriteString(orgID.String())
	b.WriteByte(':')
	b.WriteString(rowID.String())
	return []byte(b.String())
}

var ErrNotSealed = errors.New("value is not sealed")

func Seal(plaintext []byte, keyID uuid.UUID, key, aad []byte) (Sealed, error) {
	if keyID == uuid.Nil {

		return Sealed{}, errors.New("seal: key id is required")
	}
	ct, iv, tag, err := EncryptAESGCMWithAAD(plaintext, key, aad)
	if err != nil {
		return Sealed{}, fmt.Errorf("seal: %w", err)
	}
	return Sealed{
		KeyID:      keyID,
		Ciphertext: EncodeB64(ct),
		IV:         EncodeB64(iv),
		Tag:        EncodeB64(tag),
	}, nil
}

func (s Sealed) IsZero() bool {
	return s.Ciphertext == "" && s.IV == "" && s.Tag == ""
}

func (s Sealed) Open(key, aad []byte) ([]byte, error) {
	if s.IsZero() {
		return nil, ErrNotSealed
	}
	if s.Ciphertext == "" || s.IV == "" || s.Tag == "" {
		return nil, fmt.Errorf("sealed value is incomplete: ciphertext=%t iv=%t tag=%t",
			s.Ciphertext != "", s.IV != "", s.Tag != "")
	}

	ct, err1 := DecodeB64(s.Ciphertext)
	iv, err2 := DecodeB64(s.IV)
	tag, err3 := DecodeB64(s.Tag)
	if err1 != nil || err2 != nil || err3 != nil {
		return nil, errors.New("sealed value is not valid base64")
	}

	plain, err := DecryptAESGCMWithAAD(ct, key, iv, tag, aad)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	return plain, nil
}
