package fp

import (
	cryptorand "crypto/rand"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("no entropy")
}

func assertAlphanumeric(t *testing.T, s string) {
	t.Helper()
	for _, c := range s {
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		assert.True(t, ok, "unexpected character %q", c)
	}
}

func TestRandom(t *testing.T) {
	t.Run("Length", func(t *testing.T) {
		for _, n := range []int{1, 10, 64} {
			assert.Len(t, Random(n), n)
		}
	})

	t.Run("Zero Length", func(t *testing.T) {
		assert.Equal(t, "", Random(0))
	})

	t.Run("Alphanumeric Charset", func(t *testing.T) {
		assertAlphanumeric(t, Random(256))
	})

	t.Run("Unique", func(t *testing.T) {
		seen := map[string]bool{}
		for range 1000 {
			seen[Random(10)] = true
		}
		assert.Len(t, seen, 1000)
	})
}

func TestRandomPanicsOnFailure(t *testing.T) {
	randReader = failingReader{}
	defer func() { randReader = cryptorand.Reader }()

	// a dead randomness source is catastrophic: Random must panic, never fall
	// back to a predictable source for security tokens
	assert.Panics(t, func() { Random(10) })
}
