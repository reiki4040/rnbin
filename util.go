package main

import (
	"crypto/sha256"
	"encoding/hex"
)

func Sha256(bytes []byte) string {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hex.EncodeToString(hasher.Sum(nil))
}
