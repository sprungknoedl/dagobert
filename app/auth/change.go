package auth

import (
	"net/http"

	"github.com/sprungknoedl/dagobert/app/views"
	"golang.org/x/crypto/bcrypt"
)

// ChangePasswordForm renders the self-service change-password form.
// GET /auth/changepassword (registered on the secured mux — Require guarantees a user)
func (a *Auth) ChangePasswordForm(w http.ResponseWriter, r *http.Request) {
	user, err := CurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	views.ChangePassword(user).Render(r.Context(), w)
}

// ChangePassword verifies the current password and stores a new one.
// POST /auth/changepassword
func (a *Auth) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user, err := CurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	current := r.FormValue("currentPassword")
	next := r.FormValue("newPassword")
	confirm := r.FormValue("confirmPassword")

	if user.Password != "" && bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(current)) != nil {
		// TODO: add and display validaton error
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.ChangePassword(user).Render(r.Context(), w)
		return
	}
	if next == "" || next != confirm {
		// TODO: add and display validaton error
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.ChangePassword(user).Render(r.Context(), w)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	user.Password = string(hash)
	if err := a.store.SaveUser(*user); err != nil {
		http.Error(w, "failed to save user", http.StatusInternalServerError)
		return
	}

	a.session.RenewToken(r.Context())
	views.ChangePassword(user).Render(r.Context(), w)
}
