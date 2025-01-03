package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

type AuditlogCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewAuditlogCtrl(store *model.Store, acl *ACL) *AuditlogCtrl {
	return &AuditlogCtrl{store, acl}
}

func (ctrl AuditlogCtrl) List(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.store.ListAuditlog()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/auditlog-many.html", map[string]any{
		"title": "Audit Log",
		"rows":  list,
	})
}

func (ctrl AuditlogCtrl) ListForObject(w http.ResponseWriter, r *http.Request) {
	oid := r.PathValue("oid")
	list, err := ctrl.store.ListAuditlogForObject(oid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/auditlog-one.html", map[string]any{
		"title": "Audit Log",
		"rows":  list,
	})
}

func Audit(db *model.Store, r *http.Request, obj string, format string, a ...any) {
	user := GetUser(db, r)
	kase := GetCase(db, r)
	if err := db.SaveAuditlog(user, kase, obj, fmt.Sprintf(format, a...)); err != nil {
		log.Printf("|%s| %v", tty.Yellow("WARN "), err)
	}
}
