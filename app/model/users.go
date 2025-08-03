package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aarondl/authboss/v3"
	"gorm.io/gorm"
)

var _ authboss.ServerStorer = &Store{}
var _ authboss.OAuth2ServerStorer = &Store{}

var _ authboss.User = &User{}
var _ authboss.AuthableUser = &User{}
var _ authboss.OAuth2User = &User{}

// var UserRoles = []string{"Administrator", "User", "Read-Only"}
var SystemUser = User{
	ID:   "<system>",
	Name: "System",
	Role: "Administrator",
}

var ErrUserProtected = errors.New("modification to user prohibited")

type User struct {
	ID        string
	Name      string
	UPN       string
	Password  string
	Email     string
	Role      string
	LastLogin Time

	Provider     string
	AccessToken  string
	RefreshToken string
	Expiry       Time
}

// IsOAuth2User checks to see if a user was registered in the site as an oauth2 user.
func (u *User) IsOAuth2User() bool                           { return u.Provider != "" }
func (u *User) GetPID() string                               { return u.ID }
func (u *User) GetPassword() string                          { return u.Password }
func (u *User) GetOAuth2UID() (uid string)                   { return u.ID }
func (u *User) GetOAuth2Provider() (provider string)         { return u.Provider }
func (u *User) GetOAuth2AccessToken() (token string)         { return u.AccessToken }
func (u *User) GetOAuth2RefreshToken() (refreshToken string) { return u.RefreshToken }
func (u *User) GetOAuth2Expiry() (expiry time.Time)          { return time.Time(u.Expiry) }

func (u *User) PutPID(id string)                   { u.ID = id }
func (u *User) PutPassword(pw string)              { u.Password = pw }
func (u *User) PutOAuth2UID(uid string)            { u.ID = uid }
func (u *User) PutOAuth2Provider(provider string)  { u.Provider = provider }
func (u *User) PutOAuth2AccessToken(token string)  { u.AccessToken = token }
func (u *User) PutOAuth2RefreshToken(token string) { u.RefreshToken = token }
func (u *User) PutOAuth2Expiry(expiry time.Time)   { u.Expiry = Time(expiry) }

func (u *User) String() string { return fmt.Sprintf("%s (%s)", u.Name, u.UPN) }

// Load will look up the user based on the passed the PrimaryID. Under
// normal circumstances this comes from GetPID() of the user.
//
// OAuth2 logins are special-cased to return an OAuth2 pid (combination of
// provider:oauth2uid), and therefore key be special cased in a Load()
// implementation to handle that form, use ParseOAuth2PID to see
// if key is an OAuth2PID or not.
func (store *Store) Load(ctx context.Context, key string) (authboss.User, error) {
	user := &User{}
	_, uid, err := authboss.ParseOAuth2PID(key)
	if err == nil {
		err = store.DB.First(user, "id = ?", uid).Error
		if err == gorm.ErrRecordNotFound {
			return user, authboss.ErrUserNotFound
		}
		return user, err
	}

	err = store.DB.First(user, "upn = ?", key).Error
	if err == gorm.ErrRecordNotFound {
		return user, authboss.ErrUserNotFound
	}
	return user, err
}

// Save persists the user in the database, this should never
// create a user and instead return ErrUserNotFound if the user
// does not exist.
func (store *Store) Save(ctx context.Context, user authboss.User) error {
	result := store.DB.Select("*").Updates(user.(*User))
	if result.RowsAffected == 0 {
		return authboss.ErrUserNotFound
	}
	return result.Error
}

// NewFromOAuth2 should return an OAuth2User from a set
// of details returned from OAuth2Provider.FindUserDetails
// A more in-depth explanation is that once we've got an access token
// for the service in question (say a service that rhymes with book)
// the FindUserDetails function does an http request to a known endpoint
// that provides details about the user, those details are captured in a
// generic way as map[string]string and passed into this function to be
// turned into a real user.
//
// It's possible that the user exists in the database already, and so
// an attempt should be made to look that user up using the details.
// Any details that have changed should be updated. Do not save the user
// since that will be done later by OAuth2ServerStorer.SaveOAuth2()
func (store *Store) NewFromOAuth2(ctx context.Context, provider string, details map[string]string) (authboss.OAuth2User, error) {
	id := details["oid"]
	user, err := store.GetUser(id)
	if err == gorm.ErrRecordNotFound {
		user = User{ID: id}
	}

	user.Name = details["name"]
	user.UPN = details["preferred_username"]
	user.Email = details["email"]
	return &user, nil
}

// SaveOAuth2 has different semantics from the typical ServerStorer.Save,
// in this case we want to insert a user if they do not exist.
// The difference must be made clear because in the non-oauth2 case,
// we know exactly when we want to Create vs Update. However since we're
// simply trying to persist a user that may have been in our database,
// but if not should already be (since you can think of the operation as
// a caching of what's on the oauth2 provider's servers).
func (store *Store) SaveOAuth2(ctx context.Context, user authboss.OAuth2User) error {
	obj := user.(*User)
	return store.SaveUser(*obj)
}

func (store *Store) ListUsers() ([]User, error) {
	list := []User{}
	tx := store.DB.
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetUser(id string) (User, error) {
	obj := User{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveUser(obj User) error {
	if obj.ID == SystemUser.ID {
		return ErrUserProtected
	}
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteUser(id string) error {
	if id == SystemUser.ID {
		return ErrUserProtected
	}
	return store.DB.Delete(&User{}, "id = ?", id).Error
}
