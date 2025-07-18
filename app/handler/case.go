package handler

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func NewCaseCtrl(store *model.Store, acl *ACL, ts *timesketch.Client) *CaseCtrl {
	return &CaseCtrl{Ctrl: BaseCtrl{store, acl}, ts: ts}
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
	cw.Write([]string{"ID", "Name", "Severity", "Classification", "Closed", "Outcome", "Who", "What", "When", "Where", "Why", "How"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Name,
			e.Severity,
			e.Classification,
			strconv.FormatBool(e.Closed),
			e.Outcome,
			e.SummaryWho,
			e.SummaryWhat,
			e.SummaryWhen,
			e.SummaryWhere,
			e.SummaryWhy,
			e.SummaryHow,
		})
	}

	cw.Flush()
}

func (ctrl CaseCtrl) Import(w http.ResponseWriter, r *http.Request) {
	uri := "/"
	ImportCSV(ctrl.Store(), ctrl.ACL(), w, r, uri, 12, func(rec []string) {
		closed, err := strconv.ParseBool(cmp.Or(rec[4], "false"))
		if err != nil {
			Warn(w, r, err)
			return
		}

		obj := model.Case{
			ID:             fp.If(rec[0] == "", fp.Random(10), rec[0]),
			Name:           rec[1],
			Severity:       rec[2],
			Classification: rec[3],
			Closed:         closed,
			Outcome:        rec[5],
			SummaryWho:     rec[6],
			SummaryWhat:    rec[6],
			SummaryWhen:    rec[6],
			SummaryWhere:   rec[6],
			SummaryWhy:     rec[6],
			SummaryHow:     rec[6],
		}

		if err = ctrl.Store().SaveCase(obj); err != nil {
			Err(w, r, err)
			return
		}
	})
}

func (ctrl CaseCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj := model.Case{
		ID:           cid,
		SummaryWho:   "Identified actor [user/process/IP] involved in the incident",
		SummaryWhat:  "Detected [action/event] leading to [impact/artifact]",
		SummaryWhen:  "Occurred at [timestamp], duration [timeframe]",
		SummaryWhere: "Location [host/path/network] affected",
		SummaryWhy:   "Root cause [vulnerability/misconfiguration/intent] leading to incident",
		SummaryHow:   "Execution method [tool/technique/tactic] used",
	}
	if cid != "new" {
		var err error
		obj, err = ctrl.Store().GetCase(cid)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	var sketches []timesketch.Sketch
	if ctrl.ts != nil {
		sketches, _ = ctrl.ts.ListSketches()
	}

	Render(w, r, http.StatusOK, views.CasesOne(Env(ctrl, r), obj, sketches, valid.ValidationError{}))
}

func (ctrl CaseCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Case{ID: r.PathValue("cid")}
	err := Decode(ctrl.Store(), r, &dto, ValidateCase)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		var sketches []timesketch.Sketch
		if ctrl.ts != nil {
			sketches, _ = ctrl.ts.ListSketches()
		}
		Render(w, r, http.StatusUnprocessableEntity, views.CasesOne(Env(ctrl, r), dto, sketches, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.Store().SaveCase(dto); err != nil {
		Err(w, r, err)
		return
	}

	dstSummary := strings.HasSuffix(r.Referer(), "?target=summary")
	http.Redirect(w, r, fp.If(dstSummary, "/cases/"+dto.ID+"/summary/", "/cases/"), http.StatusSeeOther)
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

	http.Redirect(w, r, "/cases/", http.StatusSeeOther)
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
