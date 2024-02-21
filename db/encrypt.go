package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

type EncryptedToken = []byte

func hashKeyPhrase(keyPhrase string) string {
	hashed := md5.Sum([]byte(keyPhrase))
	return hex.EncodeToString(hashed[:])
}

func EncryptToken(token string, keyPhrase string) (EncryptedToken, error) {
	aesCipher, err := aes.NewCipher([]byte(hashKeyPhrase(keyPhrase)))
	if err != nil {
		return nil, err
	}

	gcmInstance, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcmInstance.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonce)

	ciphered := gcmInstance.Seal(nonce, nonce, []byte(token), nil)

	return ciphered, nil
}

func DecryptToken(encryptedToken EncryptedToken, keyPhrase string) (string, error) {
	aesCipher, err := aes.NewCipher([]byte(hashKeyPhrase(keyPhrase)))
	if err != nil {
		return "", err
	}

	gcmInstance, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return "", err
	}

	nonceSize := gcmInstance.NonceSize()
	nonce, encrypted := encryptedToken[:nonceSize], encryptedToken[nonceSize:]
	token, err := gcmInstance.Open(nil, nonce, encrypted, nil)

	return string(token), nil
}
