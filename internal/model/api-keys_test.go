package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyCrypto(t *testing.T) {
	plaintext, hash, hint := GenerateAPIKey()

	t.Run("a freshly generated key has a valid format", func(t *testing.T) {
		assert.True(t, ValidAPIKeyFormat(plaintext))
	})

	t.Run("a single-char mutation is rejected", func(t *testing.T) {
		mutated := []byte(plaintext)
		mutated[len(mutated)-1] ^= 1 // flip a bit in the checksum's last char
		assert.False(t, ValidAPIKeyFormat(string(mutated)))
	})

	t.Run("wrong prefix is rejected", func(t *testing.T) {
		assert.False(t, ValidAPIKeyFormat("xyz_"+plaintext[len(APIKeyPrefix):]))
	})

	t.Run("HashAPIKey is deterministic and matches the generated hash", func(t *testing.T) {
		assert.Equal(t, hash, HashAPIKey(plaintext))
		assert.NotEqual(t, plaintext, hash)
	})

	t.Run("the hint is non-secret and shows the plaintext's first 6 and last 6", func(t *testing.T) {
		assert.Equal(t, plaintext[:6]+strings.Repeat("•", len(plaintext)-12)+plaintext[len(plaintext)-6:], hint)
	})
}
