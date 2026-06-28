package handler

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/app/auth"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type CaseCtrl struct {
	Ctrl
	ts *timesketch.Client
}

func NewCaseCtrl(store *model.Store, acl *auth.ACL, ts *timesketch.Client) *CaseCtrl {
	return &CaseCtrl{Ctrl: BaseCtrl{store, acl}, ts: ts}
}

// fetchSketches loads the sketches for the case form. It reports whether the
// sketch dropdown should be shown at all (Timesketch is configured) and a
// warning when the configured instance can not be queried.
func (ctrl CaseCtrl) fetchSketches(r *http.Request) (show bool, sketches []timesketch.Sketch, warning string) {
	if !ctrl.ts.Configured() {
		return false, nil, ""
	}

	sketches, err := ctrl.ts.ListSketches(r.Context())
	if err != nil {
		return true, nil, "Failed to fetch sketches from Timesketch: " + err.Error()
	}
	return true, sketches, ""
}

func (ctrl CaseCtrl) List(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.Store().ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.CasesMany(Env(ctrl, r), list))
}

func (ctrl CaseCtrl) Export(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.Store().ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - Cases.csv", time.Now().Format("20060102"))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Name", "Severity", "Classification", "Closed", "Outcome", "Summary", "Custom"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Name,
			e.Severity,
			e.Classification,
			strconv.FormatBool(e.Closed),
			e.Outcome,
			e.Summary,
			e.Custom.JSON(),
		})
	}

	cw.Flush()
}

func (ctrl CaseCtrl) Import(w http.ResponseWriter, r *http.Request) {
	uri := "/"
	ctrl.Store().Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, ctrl.ACL(), w, r, uri, 8, func(rec []string) {
			closed, err := strconv.ParseBool(cmp.Or(rec[4], "false"))
			if err != nil {
				Warn(w, r, err)
				return
			}

			var custom model.Custom
			if len(rec) > 7 {
				custom.Scan(rec[7])
			}

			obj := model.Case{
				ID:             fp.If(rec[0] == "", fp.Random(10), rec[0]),
				Name:           rec[1],
				Severity:       rec[2],
				Classification: rec[3],
				Closed:         closed,
				Outcome:        rec[5],
				Summary:        rec[6],
				Custom:         custom,
			}

			if err = tx.SaveCase(obj); err != nil {
				Err(w, r, err)
				return
			}
		})
	})
}

func (ctrl CaseCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj := model.Case{ID: cid}

	// the "create from template" dropdown is only offered on the new-case form;
	// each template carries its defaults inline so the form fills client-side
	var templates []model.Case
	if cid == "new" {
		var err error
		templates, err = ctrl.Store().ListTemplates()
		if err != nil {
			Err(w, r, err)
			return
		}
	} else {
		var err error
		obj, err = ctrl.Store().GetCase(cid)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	show, sketches, warning := ctrl.fetchSketches(r)
	Render(w, r, http.StatusOK, views.CasesOne(Env(ctrl, r), obj, templates, "", show, sketches, warning, valid.ValidationError{}))
}

func (ctrl CaseCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Case{ID: r.PathValue("cid")}
	err := Decode(ctrl.Store(), r, &dto, ValidateCase)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		show, sketches, warning := ctrl.fetchSketches(r)
		templates, _ := ctrl.Store().ListTemplates()
		Render(w, r, http.StatusUnprocessableEntity, views.CasesOne(Env(ctrl, r), dto, templates, r.FormValue("Template"), show, sketches, warning, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// NOTE: form-only for now — CollectCustom reads r.PostForm, so a JSON API
	// request yields an empty map and won't carry custom values.
	dto.Custom = CollectCustom(r)

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.Store().SaveCase(dto); err != nil {
		Err(w, r, err)
		return
	}

	// instantiate from a template: copy its tasks and notes into the new case.
	// Only honored when creating; ignored when editing an existing case.
	if template := r.FormValue("Template"); new && template != "" {
		if _, err := ctrl.Store().CloneCaseContents(template, dto); err != nil {
			Err(w, r, err)
			return
		}
	}

	dstSummary := strings.HasSuffix(r.Referer(), "?target=summary")
	RedirectAfterSave(w, r, fp.If(dstSummary, "/cases/"+dto.ID+"/summary/", "/cases/"))
}

func (ctrl CaseCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s?confirm=yes", cid)
		views.ConfirmDialog(uri).Render(r.Context(), w)
		return
	}

	if err := ctrl.Store().DeleteCase(cid); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (ctrl CaseCtrl) EditACL(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj, err := ctrl.Store().GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	users, err := ctrl.Store().ListUsers()
	if err != nil {
		Err(w, r, err)
		return
	}

	perms, err := ctrl.Store().GetCasePermissions(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.CasesACL(Env(ctrl, r), obj, users, perms))
}

func (ctrl CaseCtrl) SaveACL(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj, err := ctrl.Store().GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	form := struct{ Users []string }{}
	if err := Decode(ctrl.Store(), r, &form, nil); err != nil {
		Warn(w, r, err)
		return
	}

	if err := ctrl.ACL().SaveCasePermissions(obj.ID, form.Users); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/cases/")
}

// Switch renders the quick case-switcher popup: the cases the user can access,
// excluding the current case and template cases, optionally narrowed by search.
// The incoming `to` section suffix is re-validated so it can't inject a path.
func (ctrl CaseCtrl) Switch(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := views.ValidSection(r.URL.Query().Get("to"))
	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))

	list, err := ctrl.Store().ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	env := Env(ctrl, r)
	cases := fp.Filter(list, func(c model.Case) bool {
		if c.ID == from || c.IsTemplate {
			return false
		}
		if _, ok := env.Allowed("GET", "/cases/"+c.ID+"/summary/"); !ok {
			return false
		}
		if search != "" {
			return strings.Contains(strings.ToLower(c.Name), search) ||
				strings.Contains(strings.ToLower(c.ID), search)
		}
		return true
	})

	Render(w, r, http.StatusOK, views.CaseSwitcher(env, cases, from, to))
}

func (ctrl CaseCtrl) Summary(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj, err := ctrl.Store().GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	assets, err := ctrl.Store().ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := ctrl.Store().ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	events, err := ctrl.Store().ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return

	}

	Render(w, r, http.StatusOK, views.CasesSummary(Env(ctrl, r), obj, events, assets, indicators))
}
