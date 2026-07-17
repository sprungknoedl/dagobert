package auth

import (
	"log/slog"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/valid"
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
	views.ChangePassword(user, nil).Render(r.Context(), w)
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

	// Field-level validation. Messages are deliberately terse: the current
	// password is never echoed back, and we do not disclose anything about the
	// stored hash beyond "it didn't match". Accounts without a local password
	// (OIDC-only) can set one without proving a previous password.
	vr := valid.ValidationError{}
	if user.Password != "" && bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(current)) != nil {
		vr["currentPassword"] = valid.Condition{Name: "currentPassword", Invalid: true, Message: "Current password is incorrect."}
	}
	if next == "" {
		vr["newPassword"] = valid.Condition{Name: "newPassword", Missing: true}
	} else if next != confirm {
		vr["confirmPassword"] = valid.Condition{Name: "confirmPassword", Invalid: true, Message: "Passwords do not match."}
	}
	if !vr.Valid() {
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.ChangePassword(user, vr).Render(r.Context(), w)
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

	if err := a.session.RenewToken(r.Context()); err != nil {
		slog.Error("failed to renew session token", "err", err)
	}

	// Close the unpoly drawer the form was submitted from. There is no list
	// page to redirect to (unlike the other drawer forms that rely on
	// up-accept-location), so accept the overlay explicitly from the server.
	// The acceptance value carries a message that dagobert.js shows as a toast.
	w.Header().Set("X-Up-Accept-Layer", `{"toast":"Your password has been changed."}`)
	w.WriteHeader(http.StatusOK)
}
