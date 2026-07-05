package handler

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

// fetchSketches loads the sketches for the case form. It reports whether the
// sketch dropdown should be shown at all (Timesketch is configured) and a
// warning when the configured instance can not be queried.
func (h *Handler) fetchSketches(r *http.Request) views.SketchInfo {
	if !h.Timesketch.Configured() {
		return views.SketchInfo{}
	}

	sketches, err := h.Timesketch.ListSketches(r.Context())
	if err != nil {
		return views.SketchInfo{Show: true, Warning: "Failed to fetch sketches from Timesketch: " + err.Error()}
	}
	return views.SketchInfo{Show: true, Sketches: sketches}
}

func (h *Handler) CaseList(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.CasesMany(h.Env(r), list))
}

func (h *Handler) CaseExport(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - Cases.csv", time.Now().Format("20060102"))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Name", "Severity", "Classification", "Closed", "Outcome", "Summary", "Opened", "Closed at", "Custom"})
	for _, e := range list {
		openedAt := fp.If(!e.OpenedAt.IsZero(), e.OpenedAt.Format("2006-01-02"), "")
		closedAt := fp.If(!e.ClosedAt.IsZero(), e.ClosedAt.Format("2006-01-02"), "")
		cw.Write([]string{
			e.ID,
			e.Name,
			e.Severity,
			e.Classification,
			strconv.FormatBool(e.Closed),
			e.Outcome,
			e.Summary,
			openedAt,
			closedAt,
			e.Custom.JSON(),
		})
	}

	cw.Flush()
}

func (h *Handler) CaseImport(w http.ResponseWriter, r *http.Request) {
	uri := "/"
	h.Store.Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, h.ACL, w, r, uri, 10, func(rec []string) {
			closed, err := strconv.ParseBool(cmp.Or(rec[4], "false"))
			if err != nil {
				Warn(w, r, err)
				return
			}

			var custom model.Custom
			custom.Scan(rec[9])

			var openedAt, closedAt model.Date
			if rec[7] != "" {
				t, err := time.Parse("2006-01-02", rec[7])
				if err != nil {
					Warn(w, r, err)
					return
				}
				openedAt = model.Date(t)
			}
			if rec[8] != "" {
				t, err := time.Parse("2006-01-02", rec[8])
				if err != nil {
					Warn(w, r, err)
					return
				}
				closedAt = model.Date(t)
			}

			if openedAt.IsZero() {
				openedAt = model.Date(time.Now())
			}
			if closed && closedAt.IsZero() {
				closedAt = model.Date(time.Now())
			} else if !closed {
				closedAt = model.Date{}
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
				OpenedAt:       openedAt,
				ClosedAt:       closedAt,
			}

			if err = tx.SaveCase(obj); err != nil {
				Err(w, r, err)
				return
			}
		})
	})
}

func (h *Handler) CaseEdit(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj := model.Case{ID: cid}

	// the "create from template" dropdown is only offered on the new-case form;
	// each template carries its defaults inline so the form fills client-side
	var templates []model.Case
	if cid == "new" {
		var err error
		templates, err = h.Store.ListTemplates()
		if err != nil {
			Err(w, r, err)
			return
		}
	} else {
		var err error
		obj, err = h.Store.GetCase(cid)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	sketches := h.fetchSketches(r)
	Render(w, r, http.StatusOK, views.CasesOne(h.Env(r), obj, valid.ValidationError{},
		views.WithTemplates(templates, ""),
		views.WithSketches(sketches)))
}

// outstandingOnClose returns human-readable, count-only messages for case
// items that are still outstanding when closing a case: open tasks,
// un-triaged assets, missing classification, and missing outcome.
//
// TODO: the "still outstanding" definitions below are hardcoded against fixed
// strings/booleans; revisit once the state/rank enum metadata can drive this
// per category instead.
func outstandingOnClose(store *model.Store, dto model.Case) ([]string, error) {
	var messages []string

	tasks, err := store.ListTasks(dto.ID)
	if err != nil {
		return nil, err
	}
	open := 0
	for _, t := range tasks {
		if !t.Done {
			open++
		}
	}
	if open > 0 {
		messages = append(messages, fmt.Sprintf("%d task%s still open", open, fp.If(open == 1, "", "s")))
	}

	assets, err := store.ListAssets(dto.ID)
	if err != nil {
		return nil, err
	}
	untriaged := 0
	for _, a := range assets {
		if a.Status == "" || a.Status == "Under investigation" {
			untriaged++
		}
	}
	if untriaged > 0 {
		messages = append(messages, fmt.Sprintf("%d asset%s still under investigation", untriaged, fp.If(untriaged == 1, "", "s")))
	}

	if dto.Classification == "" {
		messages = append(messages, "Classification not set")
	}

	if dto.Outcome == "" {
		messages = append(messages, "Outcome not set")
	}

	return messages, nil
}

func (h *Handler) CaseSave(w http.ResponseWriter, r *http.Request) {
	dto := model.Case{ID: r.PathValue("cid")}
	err := Decode(h.Store, r, &dto, ValidateCase)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		sketches := h.fetchSketches(r)
		templates, _ := h.Store.ListTemplates()
		Render(w, r, http.StatusUnprocessableEntity, views.CasesOne(h.Env(r), dto, vr,
			views.WithTemplates(templates, r.FormValue("Template")),
			views.WithSketches(sketches)))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// NOTE: form-only for now — CollectCustom reads r.PostForm, so a JSON API
	// request yields an empty map and won't carry custom values.
	dto.Custom = CollectCustom(r)

	new := dto.ID == "new"

	// Auto-fill only fills blanks — never overwrites a value the user entered.
	// Dates are irrelevant for templates, so they're left untouched there.
	if !dto.IsTemplate {
		if new && dto.OpenedAt.IsZero() {
			dto.OpenedAt = model.Date(time.Now())
		}
		if dto.Closed {
			if dto.ClosedAt.IsZero() {
				dto.ClosedAt = model.Date(time.Now())
			}
		} else {
			dto.ClosedAt = model.Date{}
		}
	}

	// Soft-confirm on an open->closed transition when outstanding items remain.
	// The user can always override via "Close anyway" (Confirm=yes); this never
	// hard-blocks the save.
	if dto.Closed && !dto.IsTemplate && r.FormValue("Confirm") != "yes" {
		wasClosed := false
		if !new {
			prior, err := h.Store.GetCase(dto.ID)
			if err != nil {
				Err(w, r, err)
				return
			}
			wasClosed = prior.Closed
		}

		if !wasClosed {
			outstanding, err := outstandingOnClose(h.Store, dto)
			if err != nil {
				Err(w, r, err)
				return
			}
			if len(outstanding) > 0 {
				sketches := h.fetchSketches(r)
				templates, _ := h.Store.ListTemplates()
				Render(w, r, http.StatusOK, views.CasesOne(h.Env(r), dto, valid.ValidationError{},
					views.WithTemplates(templates, r.FormValue("Template")),
					views.WithSketches(sketches),
					views.WithOutstanding(outstanding)))
				return
			}
		}
	}

	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveCase(dto); err != nil {
		Err(w, r, err)
		return
	}

	// instantiate from a template: copy its tasks and notes into the new case.
	// Only honored when creating; ignored when editing an existing case.
	if template := r.FormValue("Template"); new && template != "" {
		if _, err := h.Store.CloneCaseContents(template, dto); err != nil {
			Err(w, r, err)
			return
		}
	}

	dstSummary := strings.HasSuffix(r.Referer(), "?target=summary")
	RedirectAfterSave(w, r, fp.If(dstSummary, "/cases/"+dto.ID+"/summary/", "/cases/"))
}

func (h *Handler) CaseDelete(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s?confirm=yes", cid)
		views.ConfirmDialog(uri).Render(r.Context(), w)
		return
	}

	if err := h.Store.DeleteCase(cid); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) CaseEditACL(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj, err := h.Store.GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	users, err := h.Store.ListUsers()
	if err != nil {
		Err(w, r, err)
		return
	}

	perms, err := h.Store.GetCasePermissions(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.CasesACL(h.Env(r), obj, users, perms))
}

func (h *Handler) CaseSaveACL(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj, err := h.Store.GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	form := struct{ Users []string }{}
	if err := Decode(h.Store, r, &form, nil); err != nil {
		Warn(w, r, err)
		return
	}

	if err := h.ACL.SaveCasePermissions(obj.ID, form.Users); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/cases/")
}

// Switch renders the quick case-switcher popup: the cases the user can access,
// excluding the current case and template cases, optionally narrowed by search.
// The incoming `to` section suffix is re-validated so it can't inject a path.
func (h *Handler) CaseSwitch(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := views.ValidSection(r.URL.Query().Get("to"))
	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))

	list, err := h.Store.ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	env := h.Env(r)
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

func (h *Handler) CaseSummary(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj, err := h.Store.GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	assets, err := h.Store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := h.Store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	events, err := h.Store.ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return

	}

	Render(w, r, http.StatusOK, views.CasesSummary(h.Env(r), obj, events, assets, indicators))
}
