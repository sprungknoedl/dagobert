package auth

import (
	"fmt"
	"log"
	"net/http"

	"github.com/casbin/casbin/v2"
	cm "github.com/casbin/casbin/v2/model"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

type ACL struct {
	db       *model.Store
	enforcer *casbin.Enforcer
}

func NewACL(db *model.Store) *ACL {
	m := cm.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", `(g(r.sub, p.sub) || p.sub == "*") && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")`)

	enforcer, err := casbin.NewEnforcer(m, db)
	if err != nil {
		log.Fatalf("Failed to init casbin enforcer: %v \n", err.Error())
	}

	enforcer.EnableAutoSave(true)
	return &ACL{db, enforcer}
}

// Enforce decides whether a "subject" can access a "object" with the operation "action", input parameters are usually: (sub, obj, act).
func (acl *ACL) Enforce(rvals ...interface{}) (bool, error) {
	return acl.enforcer.Enforce(rvals...)
}

func (acl *ACL) Allowed(uid string, url string, method string) bool {
	ok, _ := acl.Enforce(uid, url, method)
	return ok
}

func (acl *ACL) DeleteUser(uid string) error {
	if _, err := acl.enforcer.DeleteRolesForUser(uid); err != nil {
		return err
	}
	if _, err := acl.enforcer.DeletePermissionForUser(uid); err != nil {
		return err
	}
	if _, err := acl.enforcer.DeleteUser(uid); err != nil {
		return err
	}

	return nil
}

func (acl *ACL) SaveUserRole(uid string, role string) error {
	_, err := acl.enforcer.DeleteRolesForUser(uid)
	if err != nil {
		return err
	}

	_, err = acl.enforcer.AddRoleForUser(uid, "role::"+role)
	if err != nil {
		return err
	}

	return nil
}

func (acl *ACL) SaveUserPermissions(uid string, role string, cases []string) error {
	_, err := acl.enforcer.DeletePermissionsForUser(uid)
	if err != nil {
		return err
	}

	for _, c := range cases {
		obj := fmt.Sprintf("/cases/%s/*", c)
		act := fp.If(role == "Read-Only", http.MethodGet, "*")
		_, err := acl.enforcer.AddPermissionForUser(uid, obj, act)
		if err != nil {
			return err
		}
	}

	return nil
}

func (acl *ACL) SaveCasePermissions(cid string, users []string) error {
	obj := fmt.Sprintf("/cases/%s/*", cid)
	if err := acl.db.RemoveFilteredPolicy("p", "p", 1, obj); err != nil {
		return err
	}

	for _, uid := range users {
		user, err := acl.db.GetUser(uid)
		if err != nil {
			return err
		}

		act := fp.If(user.Role == "Read-Only", http.MethodGet, "*")
		if err := acl.db.AddPolicy("p", "p", []string{user.ID, obj, act}); err != nil {
			return err
		}
	}

	if err := acl.enforcer.LoadPolicy(); err != nil {
		return err
	}

	return nil
}
