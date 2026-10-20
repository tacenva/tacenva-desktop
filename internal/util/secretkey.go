package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type encodedSecretKey struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

func EncodeSecretKey(publicKey, privateKey string) (string, error) {
	payload, err := json.Marshal(encodedSecretKey{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	})
	if err != nil {
		return "", fmt.Errorf("encode secret key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(payload), nil
}

func DecodeSecretKey(secret string) (*encodedSecretKey, error) {
	secret = strings.TrimSpace(secret)

	if secret == "" {
		return nil, fmt.Errorf("Secret Key wajib diisi")
	}

	payload, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("Secret Key tidak valid: encoding Base64 gagal")
	}

	var result encodedSecretKey

	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("Secret Key tidak valid: format data salah")
	}

	if strings.TrimSpace(result.PublicKey) == "" {
		return nil, fmt.Errorf("Secret Key tidak valid: data tidak lengkap")
	}

	if strings.TrimSpace(result.PrivateKey) == "" {
		return nil, fmt.Errorf("Secret Key tidak valid: data tidak lengkap")
	}

	return &result, nil
}
