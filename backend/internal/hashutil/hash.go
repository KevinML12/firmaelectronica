package hashutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func SHA256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	out := make([]byte, len(h))
	copy(out, h[:])
	return out
}

func SHA256Hex(b []byte) string {
	return hex.EncodeToString(SHA256Sum(b))
}

// CodigoVerificacion genera un identificador corto tipo OJ (legible en encabezado).
func CodigoVerificacion() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
