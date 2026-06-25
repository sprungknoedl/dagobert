package model

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"strings"

	"github.com/sprungknoedl/dagobert/pkg/fp"
)

const (
	// KeyPrefix is the global prefix every issued key carries. The key type
	// stays in the DB Type column rather than in a per-type prefix.
	KeyPrefix      = "dgb_"
	keyBodyLen     = 64
	keyChecksumLen = 6
)

// base62 is the alphabet used to encode the checksum, matching GitHub's scheme.
const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type Key struct {
	Key  string `gorm:"primaryKey"`
	Name string
	Type string
	Hint string
}

// GenerateKey mints a new API key. It returns the one-time plaintext shown to
// the admin, the SHA-256 hash persisted at rest, and a non-secret hint for the
// list view. The plaintext is never stored.
func GenerateKey() (plaintext, hash, hint string) {
	body := fp.Random(keyBodyLen)
	plaintext = KeyPrefix + body + keyChecksum(body)
	hash = HashKey(plaintext)
	hint = KeyPrefix + "…" + plaintext[len(plaintext)-4:]
	return plaintext, hash, hint
}

// HashKey returns the hex-encoded SHA-256 of the plaintext key.
func HashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// ValidKeyFormat reports whether plaintext has the dgb_ prefix and a checksum
// that matches its body, so a malformed/typo'd key is rejected without a DB hit.
func ValidKeyFormat(plaintext string) bool {
	if !strings.HasPrefix(plaintext, KeyPrefix) {
		return false
	}
	rest := plaintext[len(KeyPrefix):]
	if len(rest) != keyBodyLen+keyChecksumLen {
		return false
	}
	body, checksum := rest[:keyBodyLen], rest[keyBodyLen:]
	return keyChecksum(body) == checksum
}

// keyChecksum is the CRC32 (IEEE) of the body, base62-encoded and left-padded
// to keyChecksumLen characters.
func keyChecksum(body string) string {
	n := crc32.ChecksumIEEE([]byte(body))
	out := make([]byte, 0, keyChecksumLen)
	for n > 0 {
		out = append(out, base62[n%62])
		n /= 62
	}
	for len(out) < keyChecksumLen {
		out = append(out, base62[0])
	}
	// reverse for big-endian digit order
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

// KeyType binds an API key type to the principal its keys authenticate as.
type KeyType struct {
	Name      string
	Icon      string
	Principal *User
}

// KeyTypes is the code-defined registry of API key types. It replaces the old
// user-editable "KeyTypes" enum: the key form, list display, validation, and
// the api-key middleware all derive from this single source of truth.
var KeyTypes = []KeyType{
	{"API", "hio-beaker", &SystemUser},
	{"Donald", "hio-camera", &DonaldUser},
	{"MCP", "hio-cpu-chip", &McpUser},
}

// KeyTypeEnums projects the registry into []Enum for views and validation.
func KeyTypeEnums() []Enum {
	enums := make([]Enum, 0, len(KeyTypes))
	for _, kt := range KeyTypes {
		enums = append(enums, Enum{Name: kt.Name, Icon: kt.Icon})
	}
	return enums
}

// PrincipalForKeyType resolves the principal an api key of the given type
// authenticates as. The second return value is false for unknown types.
func PrincipalForKeyType(t string) (*User, bool) {
	for _, kt := range KeyTypes {
		if kt.Name == t {
			return kt.Principal, true
		}
	}
	return nil, false
}

func (store *Store) ListKeys() ([]Key, error) {
	list := []Key{}
	tx := store.DB.
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetKey(key string) (Key, error) {
	obj := Key{}
	tx := store.DB.First(&obj, "key = ?", key)
	return obj, tx.Error
}

func (store *Store) SaveKey(obj Key) error {
	return store.DB.Save(&obj).Error
}

func (store *Store) DeleteKey(key string) error {
	return store.DB.Delete(Key{}, "key = ?", key).Error
}
