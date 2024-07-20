package handler

import (
	"encoding/csv"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

var ZeroID string = "0"
var ZeroTime time.Time

var decoder = schema.NewDecoder()
var SessionName = "default"
var SessionStore = sessions.NewCookieStore([]byte(os.Getenv("WEB_SESSION_SECRET")))

func ImportCSV(store *model.Store, w http.ResponseWriter, r *http.Request, uri string, numFields int, cb func(rec []string)) {
	if r.Method == http.MethodGet {
		Render(store, w, r, http.StatusOK, "internal/views/utils-import.html", map[string]any{})
		return
	}

	fr, _, err := r.FormFile("file")
	if err != nil {
		Warn(w, r, err)
		return
	}

	cr := csv.NewReader(fr)
	cr.FieldsPerRecord = numFields
	cr.Read() // skip header

	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			Warn(w, r, err)
			return
		}

		cb(rec)
	}

	http.Redirect(w, r, uri, http.StatusSeeOther)
}

func Warn(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	log.Printf("|%s| %v", tty.Yellow(" WAR "), err)
	render(w, http.StatusBadRequest, "internal/views/toasts-warning.html", map[string]any{"err": err})
}

func Err(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	log.Printf("|%s| %v", tty.Red(" ERR "), err)
	render(w, http.StatusInternalServerError, "internal/views/toasts-error.html", map[string]any{"err": err})
}

func Render(store *model.Store, w http.ResponseWriter, r *http.Request, status int, name string, values map[string]any) {
	values["env"] = GetEnv(store, r)
	values["model"] = map[string]any{
		"AssetStatus":     model.AssetStatus,
		"AssetTypes":      model.AssetTypes,
		"CaseOutcomes":    model.CaseOutcomes,
		"CaseSeverities":  model.CaseSeverities,
		"EventTypes":      model.EventTypes,
		"EvidenceTypes":   model.EvidenceTypes,
		"IndicatorStatus": model.IndicatorStatus,
		"IndicatorTLPs":   model.IndicatorTLPs,
		"IndicatorTypes":  model.IndicatorTypes,
		"MalwareStatus":   model.MalwareStatus,
		"TaskTypes":       model.TaskTypes,
	}

	render(w, status, name, values)
}

func render(w http.ResponseWriter, status int, name string, values map[string]any) {
	tpl, err := template.New(filepath.Base(name)).Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": strings.Title,
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).ParseFiles(
		name,
		"internal/views/_layout.html",
		"internal/views/_icons.html",
	)
	if err != nil {
		log.Printf("| %s | %v", tty.Red("ERR"), err)
		return
	}

	w.WriteHeader(status)
	if err = tpl.Execute(w, values); err != nil {
		log.Printf("| %s | %v", tty.Red("ERR"), err)
		return
	}
}

func Decode(r *http.Request, dst any) error {
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			return err
		}
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	decoder.IgnoreUnknownKeys(true)
	return decoder.Decode(dst, r.PostForm)
}

type Env struct {
	Username    string
	ActiveRoute string
	ActiveCase  model.Case
	Search      string
	Sort        string
}

func GetEnv(store *model.Store, r *http.Request) Env {
	cid := r.PathValue("cid")
	kase, _ := store.GetCase(cid)

	sess, _ := SessionStore.Get(r, SessionName)
	claims, _ := sess.Values["oidcClaims"].(map[string]interface{})
	user, _ := claims["sub"].(string)

	return Env{
		Username:    user,
		ActiveRoute: r.URL.Path,
		ActiveCase:  kase,
		Search:      r.URL.Query().Get("search"),
		Sort:        r.URL.Query().Get("sort"),
	}
}

func ServeDir(prefix string, dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.StripPrefix("/dist/", fs)
}

func ServeFile(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, name)
	})
}
