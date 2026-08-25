package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("rand: %w", err)
	}
	return b, nil
}

func EncryptAESGCM(plaintext, key []byte) (cipherNoTag, iv, tag []byte, _ error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gcm: %w", err)
	}

	iv, err = RandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, nil, nil, err
	}

	cipherWithTag := gcm.Seal(nil, iv, plaintext, nil)
	if len(cipherWithTag) < 16 {
		return nil, nil, nil, errors.New("ciphertext too short")
	}

	tagLen := 16
	cipherNoTag = cipherWithTag[:len(cipherWithTag)-tagLen]
	tag = cipherWithTag[len(cipherWithTag)-tagLen:]
	return cipherNoTag, iv, tag, nil
}

func DecryptAESGCM(cipherNoTag, key, iv, tag []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	if gcm.NonceSize() != len(iv) {
		return nil, fmt.Errorf("bad nonce size: %d", len(iv))
	}

	cipherWithTag := append(append([]byte{}, cipherNoTag...), tag...)
	plain, err := gcm.Open(nil, iv, cipherWithTag, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm open: %w", err)
	}

	return plain, nil
}

func EncodeB64(b []byte) string          { return base64.StdEncoding.EncodeToString(b) }
func DecodeB64(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) }

func EncryptAESGCMWithAAD(plaintext, key, aad []byte) (cipherNoTag, iv, tag []byte, _ error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gcm: %w", err)
	}

	iv, err = RandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, nil, nil, err
	}

	cipherWithTag := gcm.Seal(nil, iv, plaintext, aad)
	if len(cipherWithTag) < 16 {
		return nil, nil, nil, errors.New("ciphertext too short")
	}

	tagLen := 16
	cipherNoTag = cipherWithTag[:len(cipherWithTag)-tagLen]
	tag = cipherWithTag[len(cipherWithTag)-tagLen:]
	return cipherNoTag, iv, tag, nil
}

func DecryptAESGCMWithAAD(cipherNoTag, key, iv, tag, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	if gcm.NonceSize() != len(iv) {
		return nil, fmt.Errorf("bad nonce size: %d", len(iv))
	}

	cipherWithTag := append(append([]byte{}, cipherNoTag...), tag...)
	plain, err := gcm.Open(nil, iv, cipherWithTag, aad)
	if err != nil {
		return nil, fmt.Errorf("gcm open: %w", err)
	}

	return plain, nil
}
