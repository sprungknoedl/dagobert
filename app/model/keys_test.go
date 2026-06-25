package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyCrypto(t *testing.T) {
	plaintext, hash, hint := GenerateKey()

	t.Run("a freshly generated key has a valid format", func(t *testing.T) {
		assert.True(t, ValidKeyFormat(plaintext))
	})

	t.Run("a single-char mutation is rejected", func(t *testing.T) {
		mutated := []byte(plaintext)
		mutated[len(mutated)-1] ^= 1 // flip a bit in the checksum's last char
		assert.False(t, ValidKeyFormat(string(mutated)))
	})

	t.Run("wrong prefix is rejected", func(t *testing.T) {
		assert.False(t, ValidKeyFormat("xyz_"+plaintext[len(KeyPrefix):]))
	})

	t.Run("HashKey is deterministic and matches the generated hash", func(t *testing.T) {
		assert.Equal(t, hash, HashKey(plaintext))
		assert.NotEqual(t, plaintext, hash)
	})

	t.Run("the hint is non-secret and ends with the plaintext's last 4", func(t *testing.T) {
		assert.Equal(t, KeyPrefix+"…"+plaintext[len(plaintext)-4:], hint)
	})
}
