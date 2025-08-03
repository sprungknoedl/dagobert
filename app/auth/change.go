package auth

import (
	"log/slog"
	"net/http"

	"github.com/aarondl/authboss/v3"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
)

func init() {
	m := &ChangePassword{}
	authboss.RegisterModule("changepassword", m)
}

// ChangePassword module
type ChangePassword struct {
	*authboss.Authboss
}

// Init module
func (m *ChangePassword) Init(ab *authboss.Authboss) (err error) {
	m.Authboss = ab

	m.Config.Core.Router.Get("/changepassword", m.Core.ErrorHandler.Wrap(m.Get))
	m.Config.Core.Router.Post("/changepassword", m.Core.ErrorHandler.Wrap(m.Post))
	return nil
}

func (m *ChangePassword) Get(w http.ResponseWriter, r *http.Request) error {
	user, err := ab.CurrentUser(r)
	if err != nil {
		slog.Warn("failed to get current user for change password action", "err", err)
		http.Redirect(w, r, ab.Paths.NotAuthorized, http.StatusSeeOther)
	}

	return views.ChangePassword(user.(*model.User)).Render(r.Context(), w)
}

func (m *ChangePassword) Post(w http.ResponseWriter, r *http.Request) error {
	user, err := ab.CurrentUser(r)
	if err != nil {
		return err
	}

	err = ab.VerifyPassword(user.(authboss.AuthableUser), "")
	if err != nil {
		return err
	}

	err = ab.UpdatePassword(r.Context(), user.(authboss.AuthableUser), "")
	if err != nil {
		return err
	}

	return nil
}
