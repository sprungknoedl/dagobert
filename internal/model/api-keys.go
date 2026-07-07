package model

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"strings"

	"github.com/sprungknoedl/dagobert/pkg/fp"
)

const (
	// APIKeyPrefix is the global prefix every issued key carries. The key type
	// stays in the DB Type column rather than in a per-type prefix.
	APIKeyPrefix   = "dgb_"
	keyBodyLen     = 64
	keyChecksumLen = 6
)

// base62 is the alphabet used to encode the checksum, matching GitHub's scheme.
const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type APIKey struct {
	Key  string `gorm:"primaryKey"`
	Name string
	Type string
	Hint string
}

// GenerateAPIKey mints a new API key. It returns the one-time plaintext shown to
// the admin, the SHA-256 hash persisted at rest, and a non-secret hint for the
// list view. The plaintext is never stored.
func GenerateAPIKey() (plaintext, hash, hint string) {
	body := fp.Random(keyBodyLen)
	plaintext = APIKeyPrefix + body + keyChecksum(body)
	hash = HashAPIKey(plaintext)
	hint = plaintext[:6] + strings.Repeat("•", len(plaintext)-12) + plaintext[len(plaintext)-6:]
	return plaintext, hash, hint
}

// HashAPIKey returns the hex-encoded SHA-256 of the plaintext key.
func HashAPIKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// ValidAPIKeyFormat reports whether plaintext has the dgb_ prefix and a checksum
// that matches its body, so a malformed/typo'd key is rejected without a DB hit.
func ValidAPIKeyFormat(plaintext string) bool {
	if !strings.HasPrefix(plaintext, APIKeyPrefix) {
		return false
	}
	rest := plaintext[len(APIKeyPrefix):]
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

// APIKeyType binds an API key type to the principal its keys authenticate as.
type APIKeyType struct {
	Name      string
	Icon      string
	Principal *User
}

// APIKeyTypes is the code-defined registry of API key types. It replaces the old
// user-editable "KeyTypes" value list: the key form, list display, validation, and
// the api-key middleware all derive from this single source of truth.
var APIKeyTypes = []APIKeyType{
	{"API", "ph-flask", &SystemUser},
	{"Donald", "ph-bird", &DonaldUser},
	{"MCP", "ph-cpu", &McpUser},
}

// APIKeyTypeValues projects the registry into []ValueListItem for views and validation.
func APIKeyTypeValues() []ValueListItem {
	valueLists := make([]ValueListItem, 0, len(APIKeyTypes))
	for _, kt := range APIKeyTypes {
		valueLists = append(valueLists, ValueListItem{Name: kt.Name, Icon: kt.Icon})
	}
	return valueLists
}

// PrincipalForAPIKeyType resolves the principal an api key of the given type
// authenticates as. The second return value is false for unknown types.
func PrincipalForAPIKeyType(t string) (*User, bool) {
	for _, kt := range APIKeyTypes {
		if kt.Name == t {
			return kt.Principal, true
		}
	}
	return nil, false
}

func (store *Store) ListAPIKeys() ([]APIKey, error) {
	list := []APIKey{}
	tx := store.DB.
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetAPIKey(key string) (APIKey, error) {
	obj := APIKey{}
	tx := store.DB.First(&obj, "key = ?", key)
	return obj, tx.Error
}

func (store *Store) SaveAPIKey(obj APIKey) error {
	return store.DB.Save(&obj).Error
}

func (store *Store) DeleteAPIKey(key string) error {
	return store.DB.Delete(APIKey{}, "key = ?", key).Error
}
