package fp

import (
	cryptorand "crypto/rand"
	"io"
	mathrand "math/rand"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// randReader is the source of randomness for Random. It is a variable so
// tests can exercise the math/rand fallback path.
var randReader io.Reader = cryptorand.Reader

// Random returns a random alphanumeric string of length n. It draws from
// crypto/rand and silently falls back to a time-seeded math/rand source if
// the system randomness source is unavailable.
func Random(n int) string {
	if out, err := randomCrypto(n); err == nil {
		return out
	}
	return randomMath(n)
}

func randomCrypto(n int) (string, error) {
	b := make([]byte, 0, n)
	buf := make([]byte, max(n, 1))
	for len(b) < n {
		if _, err := io.ReadFull(randReader, buf); err != nil {
			return "", err
		}
		for _, c := range buf {
			// rejection sampling to avoid modulo bias
			if idx := int(c & letterIdxMask); idx < len(letterBytes) {
				b = append(b, letterBytes[idx])
				if len(b) == n {
					break
				}
			}
		}
	}
	return string(b), nil
}

func randomMath(n int) string {
	var src = mathrand.NewSource(time.Now().UnixNano())

	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
