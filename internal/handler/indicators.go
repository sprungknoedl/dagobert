package handler

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/openioc"
	"github.com/sprungknoedl/dagobert/pkg/stix"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) IndicatorList(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	enrichments, err := h.Store.ListEnrichmentsForCase(cid, "Indicator")
	if err != nil {
		Err(w, r, err)
		return
	}

	comments, err := h.Store.CountComments(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.IndicatorsMany(h.Env(r), list, enrichments, comments), list)
}

func (h *Handler) IndicatorExportCSV(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(h.Store, r)
	filename := fmt.Sprintf("%s - %s - Indicators.csv", time.Now().Format("20060102"), kase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Status", "Type", "Value", "TLP", "Source", "Notes", "Custom"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Status,
			e.Type,
			e.Value,
			e.TLP,
			e.Source,
			e.Notes,
			e.Custom.JSON(),
		})
	}

	cw.Flush()
}

func (h *Handler) IndicatorExportOpenIOC(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Indicators.ioc", time.Now().Format("20060102"), GetCase(h.Store, r).Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	export := buildOpenIOC(list, GetUser(r).Name, time.Now())

	xw := xml.NewEncoder(w)
	xw.Encode(export)
	xw.Flush()
}

// buildOpenIOC maps a list of indicators into an OpenIOC 1.1 document. The
// pkg/openioc package owns the format; this function owns the Dagobert-specific
// indicator-type to OpenIOC context mapping.
func buildOpenIOC(list []model.Indicator, author string, now time.Time) *openioc.Document {
	doc := openioc.New(author, now)
	for _, ioc := range list {
		switch ioc.Type {
		case "IP":
			doc.AddItem("is", openioc.Context{Document: "PortItem", Search: "PortItem/RemoteIP", Type: "mir"}, "IP", ioc.Value)
		case "Domain":
			doc.AddItem("contains", openioc.Context{Document: "DnsEntryItem", Search: "DnsEntryItem/Host", Type: "mir"}, "string", ioc.Value)
		case "URL":
			doc.AddItem("contains", openioc.Context{Document: "Network", Search: "Network/URI", Type: "mir"}, "string", ioc.Value)
		case "Path":
			doc.AddItem("contains", openioc.Context{Document: "FileItem", Search: "FileItem/FileFullPath", Type: "mir"}, "string", ioc.Value)
		case "Hash":
			switch len(ioc.Value) {
			case 32: // MD5
				doc.AddItem("is", openioc.Context{Document: "FileItem", Search: "FileItem/Md5sum", Type: "mir"}, "string", ioc.Value)
			case 40: // SHA1
				doc.AddItem("is", openioc.Context{Document: "FileItem", Search: "FileItem/Sha1sum", Type: "mir"}, "string", ioc.Value)
			case 64: // SHA256
				doc.AddItem("is", openioc.Context{Document: "FileItem", Search: "FileItem/Sha256sum", Type: "mir"}, "string", ioc.Value)
			default: // Unknown hash
				doc.AddItem("is", openioc.Context{Document: "Other", Search: "FileItem/Other", Type: "mir"}, "string", ioc.Value)
			}
		case "Service":
			doc.AddItem("is", openioc.Context{Document: "ServiceItem", Search: "ServiceItem/Name", Type: "mir"}, "string", ioc.Value)
		default:
			doc.AddItem("is", openioc.Context{Document: "Other", Search: "Other/Other", Type: "mir"}, "string", ioc.Value)
		}
	}

	return doc
}

func (h *Handler) IndicatorExportStix(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Indicators.stix", time.Now().Format("20060102"), GetCase(h.Store, r).Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	export := buildStixBundle(list, time.Now())

	jw := json.NewEncoder(w)
	jw.Encode(export)
}

// buildStixBundle maps a list of indicators into a STIX 2.1 bundle. The pkg/stix
// package owns the format; this function owns the Dagobert-specific
// indicator-type to STIX pattern mapping.
func buildStixBundle(list []model.Indicator, now time.Time) *stix.Bundle {
	b := stix.NewBundle()
	for _, ioc := range list {
		v := stix.QuoteLiteral(ioc.Value)
		switch ioc.Type {
		case "IP":
			b.AddIndicator(fmt.Sprintf("[ipv4-addr:value='%s']", v), now)
		case "Domain":
			b.AddIndicator(fmt.Sprintf("[domain-name:value='%s']", v), now)
		case "URL":
			b.AddIndicator(fmt.Sprintf("[url:value='%s']", v), now)
		case "Path":
			b.AddIndicator(fmt.Sprintf("[directory:path='%s' AND file:name='%s']", stix.QuoteLiteral(filepath.Dir(ioc.Value)), stix.QuoteLiteral(filepath.Base(ioc.Value))), now)
		case "Hash":
			switch len(ioc.Value) {
			case 32: // MD5
				b.AddIndicator(fmt.Sprintf("[file:hashes.MD5='%s']", v), now)
			case 40: // SHA1 — hash key contains a hyphen and must be quoted
				b.AddIndicator(fmt.Sprintf("[file:hashes.'SHA-1'='%s']", v), now)
			case 64: // SHA256 — hash key contains a hyphen and must be quoted
				b.AddIndicator(fmt.Sprintf("[file:hashes.'SHA-256'='%s']", v), now)
			default: // Unknown hash
				b.AddIndicator(fmt.Sprintf("[file:hashes.Other='%s']", v), now)
			}
		case "Service":
			b.AddIndicator(fmt.Sprintf("[process:extensions.'windows-service-ext'.service_name='%s']", v), now)
		default:
			b.AddIndicator(fmt.Sprintf("[x-dagobert:value='%s']", v), now)
		}
	}

	return b
}

func (h *Handler) IndicatorImportCSV(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/indicators/", cid)
	h.Store.Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, h.ACL, w, r, uri, 8, func(rec []string) {
			var custom model.Custom
			if len(rec) > 7 {
				custom.Scan(rec[7])
			}

			obj := model.Indicator{
				ID:     fp.If(rec[0] == "", fp.Random(10), rec[0]),
				Status: rec[1],
				Type:   rec[2],
				Value:  refang(rec[3]),
				TLP:    rec[4],
				Source: rec[5],
				Notes:  rec[6],
				CaseID: cid,
				Custom: custom,
			}

			if err := tx.SaveIndicator(cid, obj, true); err != nil {
				Err(w, r, err)
				return
			}
		})
	})
}

func (h *Handler) IndicatorImportTimesketch(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	kase, err := h.Store.GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	if !h.Timesketch.Configured() {
		Warn(w, r, errors.New("timesketch integration is not configured"))
		return
	}
	if kase.SketchID == 0 {
		Warn(w, r, errors.New("case is not linked to a Timesketch sketch"))
		return
	}

	sketch, err := h.Timesketch.GetSketch(r.Context(), kase.SketchID)
	if err != nil {
		Warn(w, r, err)
		return
	}

	if err := saveTimesketchIndicators(h.Store, cid, sketch.Attributes["intelligence"].Values.Data); err != nil {
		Err(w, r, err)
		return
	}

	uri := fmt.Sprintf("/cases/%s/indicators/", cid)
	http.Redirect(w, r, uri, http.StatusSeeOther)
}

// tsIndicatorTypes maps Timesketch intelligence types to dagobert indicator types.
var tsIndicatorTypes = map[string]string{
	"fs_path":     "Path",
	"hostname":    "Domain",
	"ipv4":        "IP",
	"hash_sha256": "Hash",
	"hash_sha1":   "Hash",
	"hash_md5":    "Hash",
	"other":       "Other",
}

// saveTimesketchIndicators maps a sketch's intelligence attributes to case
// indicators and saves them in one transaction, so a failed import does not
// leave a partial batch behind.
func saveTimesketchIndicators(store *model.Store, cid string, values []timesketch.Intelligence) error {
	return store.Transaction(func(tx *model.Store) error {
		for _, value := range values {
			obj := model.Indicator{
				ID:     fp.Random(10),
				CaseID: cid,
				Type:   tsIndicatorTypes[value.Type],
				Value:  value.IOC,
				Source: "timesketch",
				Status: "Under investigation",
				TLP:    "TLP:RED",
			}

			if err := tx.SaveIndicator(cid, obj, false); err != nil {
				return err
			}
		}
		return nil
	})
}

func (h *Handler) IndicatorEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Indicator{ID: id, CaseID: cid}
	overlap := views.IndicatorOverlap{}
	enrichments := []model.Enrichment{}
	if id != "new" {
		var err error
		obj, err = h.Store.GetIndicator(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}

		refs, err := h.Store.ListIndicatorOverlap(cid, obj.Type, obj.Value)
		if err != nil {
			Err(w, r, err)
			return
		}

		user := GetUser(r)
		for _, c := range refs {
			if h.ACL.Allowed(user.ID, fmt.Sprintf("/cases/%s/", c.ID), "GET") {
				overlap.Cases = append(overlap.Cases, c)
			}
		}
		overlap.Hidden = len(refs) - len(overlap.Cases)

		enrichments, err = h.Store.ListEnrichments("Indicator", id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.IndicatorsOne(h.Env(r), obj, overlap, enrichments, valid.ValidationError{}), obj)
}

func (h *Handler) IndicatorSave(w http.ResponseWriter, r *http.Request) {
	dto := model.Indicator{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	err := Decode(h.Store, r, &dto, ValidateIndicator)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.IndicatorsOne(h.Env(r), dto, views.IndicatorOverlap{}, nil, vr), vr)
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
	dto.Value = refang(dto.Value)
	if err := h.Store.SaveIndicator(dto.CaseID, dto, true); err != nil {
		Err(w, r, err)
		return
	}

	// trigger registered automation rules
	if new {
		modules.TriggerOnIndicatorAdded(h.Store, dto)
	}

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/indicators/", dto.CaseID), dto)
}

func (h *Handler) IndicatorDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" && !wantsJSON(r) {
		uri := fmt.Sprintf("/cases/%s/indicators/%s?confirm=yes", cid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	err := h.Store.DeleteIndicator(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	if wantsJSON(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/indicators/", cid), http.StatusSeeOther)
}

func (h *Handler) IndicatorListModules(w http.ResponseWriter, r *http.Request) {
	ListModules(h, w, r, h.Store.GetIndicator)
}

func (h *Handler) IndicatorScheduleModule(w http.ResponseWriter, r *http.Request) {
	ScheduleModule(h, w, r, h.Store.GetIndicator)
}

// Removes any defanging done to indicator values.
func refang(ioc string) string {
	translate := map[string]string{
		"[.]":    ".",
		"[:]":    ":",
		"[://]":  "://",
		"hxxp:":  "http:",
		"hxxps:": "https:",
		"sfxp:":  "sftp:",
		"fxp:":   "ftp:",
		"fxle:":  "file:",
	}

	for old, new := range translate {
		ioc = strings.ReplaceAll(ioc, old, new)
	}
	return ioc
}
