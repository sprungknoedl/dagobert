package model

type Key struct {
	Key  string `gorm:"primaryKey"`
	Name string
	Type string
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
