package handler

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type IndicatorCtrl struct {
	Ctrl
	ts *timesketch.Client
}

func NewIndicatorCtrl(store *model.Store, acl *ACL, ts *timesketch.Client) *IndicatorCtrl {
	return &IndicatorCtrl{Ctrl: BaseCtrl{store, acl}, ts: ts}
}

func (ctrl IndicatorCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.IndicatorsMany(Env(ctrl, r), list))
}

func (ctrl IndicatorCtrl) ExportCSV(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(ctrl.Store(), r)
	filename := fmt.Sprintf("%s - %s - Indicators.csv", time.Now().Format("20060102"), kase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Status", "Type", "Value", "TLP", "Source", "Notes"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Status,
			e.Type,
			e.Value,
			e.TLP,
			e.Source,
			e.Notes,
		})
	}

	cw.Flush()
}

func (ctrl IndicatorCtrl) ExportOpenIOC(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Indicators.ioc", time.Now().Format("20060102"), GetCase(ctrl.Store(), r).Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	export := OpenIOC{
		Metadata: OpenIOCMetadata{
			AuthoredBy:   GetUser(ctrl.Store(), r).Name,
			AuthoredDate: time.Now(),
		},
		Criteria: []OpenIOCIndicator{{
			ID:       uuid.NewString(),
			Operator: "OR",
		}},
	}

	//var IndicatorTypes = FromEnv("VALUES_INDICATOR_TYPES", []string{"IP", "Domain", "URL", "Path", "Hash", "Service", "Other"})
	for _, ioc := range list {
		switch ioc.Type {
		case "IP":
			export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
				Condition: "is",
				ID:        uuid.NewString(),
				Context:   OpenIOCContext{Document: "PortItem", Search: "PortItem/RemoteIP", Type: "mir"},
				Content:   OpenIOCContent{Type: "IP", Value: ioc.Value},
			})
		case "Domain":
			export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
				Condition: "contains",
				ID:        uuid.NewString(),
				Context:   OpenIOCContext{Document: "DnsEntryItem", Search: "DnsEntryItem/Host", Type: "mir"},
				Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
			})
		case "URL":
			export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
				Condition: "contains",
				ID:        uuid.NewString(),
				Context:   OpenIOCContext{Document: "Network", Search: "Network/URI", Type: "mir"},
				Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
			})
		case "Path":
			export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
				Condition: "contains",
				ID:        uuid.NewString(),
				Context:   OpenIOCContext{Document: "FileItem", Search: "FileItem/FileFullPath", Type: "mir"},
				Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
			})
		case "Hash":
			if len(ioc.Value) == 32 { // MD5
				export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
					Condition: "is",
					ID:        uuid.NewString(),
					Context:   OpenIOCContext{Document: "FileItem", Search: "FileItem/Md5sum", Type: "mir"},
					Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
				})
			} else if len(ioc.Value) == 40 { // SHA1
				export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
					Condition: "is",
					ID:        uuid.NewString(),
					Context:   OpenIOCContext{Document: "FileItem", Search: "FileItem/Sha1sum", Type: "mir"},
					Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
				})
			} else if len(ioc.Value) == 64 { // SHA256
				export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
					Condition: "is",
					ID:        uuid.NewString(),
					Context:   OpenIOCContext{Document: "FileItem", Search: "FileItem/Sha256sum", Type: "mir"},
					Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
				})
			} else { // Unknown hash
				export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
					Condition: "is",
					ID:        uuid.NewString(),
					Context:   OpenIOCContext{Document: "Other", Search: "FileItem/Other", Type: "mir"},
					Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
				})
			}
		case "Service":
			export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
				Condition: "is",
				ID:        uuid.NewString(),
				Context:   OpenIOCContext{Document: "ServiceItem", Search: "ServiceItem/Name", Type: "mir"},
				Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
			})
		case "Other":
			export.Criteria[0].Items = append(export.Criteria[0].Items, OpenIOCIndicatorItem{
				Condition: "is",
				ID:        uuid.NewString(),
				Context:   OpenIOCContext{Document: "Other", Search: "Other/Other", Type: "mir"},
				Content:   OpenIOCContent{Type: "string", Value: ioc.Value},
			})
		}
	}

	xw := xml.NewEncoder(w)
	xw.Encode(export)
	xw.Flush()
}

func (ctrl IndicatorCtrl) ExportStix(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Indicators.stix", time.Now().Format("20060102"), GetCase(ctrl.Store(), r).Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	export := StixBundle{
		ID:   "bundle--" + uuid.NewString(),
		Type: "bundle",
	}

	//var IndicatorTypes = FromEnv("VALUES_INDICATOR_TYPES", []string{"IP", "Domain", "URL", "Path", "Hash", "Service", "Other"})
	for _, ioc := range list {
		switch ioc.Type {
		case "IP":
			export.Objects = append(export.Objects, StixIndicator{
				Type:        "indicator",
				Pattern:     fmt.Sprintf("[ipv4-addr:value='%s']", ioc.Value),
				PatternType: "stix",
				ValidFrom:   time.Now(),
			})
		case "Domain":
			export.Objects = append(export.Objects, StixIndicator{
				Type:        "indicator",
				Pattern:     fmt.Sprintf("[domain-name:value='%s']", ioc.Value),
				PatternType: "stix",
				ValidFrom:   time.Now(),
			})
		case "URL":
			export.Objects = append(export.Objects, StixIndicator{
				Type:        "indicator",
				Pattern:     fmt.Sprintf("[url:value='%s']", ioc.Value),
				PatternType: "stix",
				ValidFrom:   time.Now(),
			})
		case "Path":
			export.Objects = append(export.Objects, StixIndicator{
				Type:        "indicator",
				Pattern:     fmt.Sprintf("[directory:path='%s' AND file:name='%s']", filepath.Dir(ioc.Value), filepath.Base(ioc.Value)),
				PatternType: "stix",
				ValidFrom:   time.Now(),
			})
		case "Hash":
			if len(ioc.Value) == 32 { // MD5
				export.Objects = append(export.Objects, StixIndicator{
					Type:        "indicator",
					Pattern:     fmt.Sprintf("[file:hashes.MD5='%s']", ioc.Value),
					PatternType: "stix",
					ValidFrom:   time.Now(),
				})
			} else if len(ioc.Value) == 40 { // SHA1
				export.Objects = append(export.Objects, StixIndicator{
					Type:        "indicator",
					Pattern:     fmt.Sprintf("[file:hashes.SHA-1='%s']", ioc.Value),
					PatternType: "stix",
					ValidFrom:   time.Now(),
				})
			} else if len(ioc.Value) == 64 { // SHA256
				export.Objects = append(export.Objects, StixIndicator{
					Type:        "indicator",
					Pattern:     fmt.Sprintf("[file:hashes.SHA-256='%s']", ioc.Value),
					PatternType: "stix",
					ValidFrom:   time.Now(),
				})
			} else { // Unknown hash
				export.Objects = append(export.Objects, StixIndicator{
					Type:        "indicator",
					Pattern:     fmt.Sprintf("[file:hashes.Other='%s']", ioc.Value),
					PatternType: "stix",
					ValidFrom:   time.Now(),
				})
			}
		case "Service":
			export.Objects = append(export.Objects, StixIndicator{
				Type:        "indicator",
				Pattern:     fmt.Sprintf("[windows-service-ext:service_name='%s']", ioc.Value),
				PatternType: "stix",
				ValidFrom:   time.Now(),
			})
		case "Other":
			export.Objects = append(export.Objects, StixIndicator{
				Type:        "indicator",
				Pattern:     fmt.Sprintf("[other='%s']", ioc.Value),
				PatternType: "stix",
				ValidFrom:   time.Now(),
			})
		}
	}

	jw := json.NewEncoder(w)
	jw.Encode(export)
}

func (ctrl IndicatorCtrl) ImportCSV(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/indicators/", cid)
	ImportCSV(ctrl.Store(), ctrl.ACL(), w, r, uri, 7, func(rec []string) {
		obj := model.Indicator{
			ID:     fp.If(rec[0] == "", fp.Random(10), rec[0]),
			Status: rec[1],
			Type:   rec[2],
			Value:  rec[3],
			TLP:    rec[4],
			Source: rec[5],
			Notes:  rec[6],
			CaseID: cid,
		}

		if err := ctrl.Store().SaveIndicator(cid, obj, true); err != nil {
			Err(w, r, err)
			return
		}
	})
}

func (ctrl IndicatorCtrl) ImportTimesketch(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	kase, err := ctrl.Store().GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	if ctrl.ts == nil || kase.SketchID == 0 {
		Err(w, r, errors.New("invalid timesketch configuration"))
		return
	}

	sketch, err := ctrl.ts.GetSketch(kase.SketchID)
	if err != nil {
		Err(w, r, err)
		return
	}

	for _, value := range sketch.Attributes["intelligence"].Values.Data {
		lookup := map[string]string{
			"fs_path":     "Path",
			"hostname":    "Domain",
			"ipv4":        "IP",
			"hash_sha256": "Hash",
			"hash_sha1":   "Hash",
			"hash_md5":    "Hash",
			"other":       "Other",
		}

		obj := model.Indicator{
			ID:     fp.Random(10),
			CaseID: cid,
			Type:   lookup[value.Type],
			Value:  value.IOC,
			Source: "timesketch",
			Status: "Under investigation",
			TLP:    "TLP:RED",
		}

		if err = ctrl.Store().SaveIndicator(cid, obj, false); err != nil {
			Err(w, r, err)
			return
		}
	}

	uri := fmt.Sprintf("/cases/%s/indicators/", cid)
	http.Redirect(w, r, uri, http.StatusSeeOther)
}

func (ctrl IndicatorCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Indicator{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.Store().GetIndicator(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.IndicatorsOne(Env(ctrl, r), obj, valid.Result{}))
}

func (ctrl IndicatorCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Indicator{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	enums, err := ctrl.Store().ListEnums()
	if err != nil {
		Err(w, r, err)
		return
	}

	if vr := ValidateIndicator(dto, enums); !vr.Valid() {
		Render(w, r, http.StatusUnprocessableEntity, views.IndicatorsOne(Env(ctrl, r), dto, vr))
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.Store().SaveIndicator(dto.CaseID, dto, true); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/indicators/", dto.CaseID), http.StatusSeeOther)
}

func (ctrl IndicatorCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/indicators/%s?confirm=yes", cid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	err := ctrl.Store().DeleteIndicator(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/indicators/", cid), http.StatusSeeOther)
}

// Types for OpenIOC export
type OpenIOC struct {
	XMLName       xml.Name  `xml:"OpenIOC"`
	Namespace     string    `xml:"xmlns,attr"`
	ID            string    `xml:"id,attr"`
	LastModified  time.Time `xml:"last-modified,attr"`
	PublishedDate time.Time `xml:"published-date,attr"`

	Metadata OpenIOCMetadata    `xml:"metadata"`
	Criteria []OpenIOCIndicator `xml:"criteria>Indicator"`
}

type OpenIOCMetadata struct {
	ShortDescription string
	Keywords         string
	AuthoredBy       string
	AuthoredDate     time.Time
}

type OpenIOCIndicator struct {
	ID       string                 `xml:"id,attr"`
	Operator string                 `xml:"operator,attr"`
	Items    []OpenIOCIndicatorItem `xml:"IndicatorItem"`
}

type OpenIOCIndicatorItem struct {
	ID        string         `xml:"id,attr"`
	Condition string         `xml:"condition,attr"`
	Context   OpenIOCContext `xml:"Context"`
	Content   OpenIOCContent `xml:"Content"`
}

type OpenIOCContext struct {
	Document string `xml:"document,attr"`
	Search   string `xml:"search,attr"`
	Type     string `xml:"type,attr"`
}

type OpenIOCContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}

// Types for STIX export
type StixBundle struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Objects []StixIndicator `json:"objects"`
}

type StixIndicator struct {
	Type        string    `json:"type"`
	Pattern     string    `json:"pattern"`
	PatternType string    `json:"pattern_type"`
	ValidFrom   time.Time `json:"valid_from"`
}
