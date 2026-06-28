package fp

import (
	cryptorand "crypto/rand"
	"fmt"
	"io"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
)

// randReader is the source of randomness for Random. It is a variable so tests
// can exercise the failure path.
var randReader io.Reader = cryptorand.Reader

// Random returns a random alphanumeric string of length n, drawn from
// crypto/rand. Since Go 1.24 crypto/rand reads from a dedicated OS API
// (getrandom on Linux, arc4random_buf on macOS) and cannot return an error in
// normal operation — the runtime crashes the process if the OS RNG genuinely
// fails. A read error therefore means catastrophic system failure, so Random
// panics rather than silently degrading to a predictable source: these values
// back OIDC state, archive staging tokens and API key bodies.
func Random(n int) string {
	b := make([]byte, 0, n)
	buf := make([]byte, max(n, 1))
	for len(b) < n {
		if _, err := io.ReadFull(randReader, buf); err != nil {
			panic(fmt.Sprintf("fp.Random: system randomness source unavailable: %v", err))
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
	return string(b)
}
