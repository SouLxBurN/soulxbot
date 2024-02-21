package db

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	token := "this-is-my-token"
	passPhrase := "my-passphrase"

	encrypted, err := EncryptToken(token, passPhrase)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := DecryptToken(encrypted, passPhrase)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, token, decrypted)
}
