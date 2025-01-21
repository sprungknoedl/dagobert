package handler

import (
	"cmp"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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

func ImportCSV(store *model.Store, acl *ACL, w http.ResponseWriter, r *http.Request, uri string, numFields int, cb func(rec []string)) {
	if r.Method == http.MethodGet {
		Render(store, acl, w, r, http.StatusOK, "internal/views/utils-import.html", map[string]any{})
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
	render(w, r, http.StatusBadRequest, "internal/views/toasts-warning.html", map[string]any{"err": err})
}

func Err(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	log.Printf("|%s| %v", tty.Red(" ERR "), err)
	render(w, r, http.StatusInternalServerError, "internal/views/toasts-error.html", map[string]any{"err": err})
}

func Render(store *model.Store, acl *ACL, w http.ResponseWriter, r *http.Request, status int, name string, values map[string]any) {
	values["acl"] = acl
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
		"UserRoles":       model.UserRoles,
	}

	render(w, r, status, name, values)
}

func render(w http.ResponseWriter, r *http.Request, status int, name string, values map[string]any) {
	// JSON encodes one of the keys 'rows', 'obj' or 'err' in values as json
	// if requested by the client.
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		if v, ok := values["valid"]; ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(v)
			return

		} else if v, ok := values["rows"]; ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(v)
			return

		} else if v, ok := values["obj"]; ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(v)
			return

		} else if v, ok := values["err"].(error); ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"error": v.Error()})
			return
		}
	}

	tpl, err := template.New(filepath.Base(name)).Funcs(template.FuncMap{
		"lower":    strings.ToLower,
		"upper":    strings.ToUpper,
		"title":    strings.Title,
		"contains": slices.Contains[[]string],
		"json": func(value any) string {
			out, _ := json.Marshal(value)
			return string(out)
		},
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
		"allowed": func(url, method string) bool {
			acl, ok1 := values["acl"].(*ACL)
			env, ok2 := values["env"].(Env)
			if !ok1 || !ok2 || acl == nil {
				return false
			}

			res := acl.Allowed(env.UID, url, method)
			return res
		},
	}).ParseFiles(
		name,
		"internal/views/_layout.html",
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
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return json.NewDecoder(r.Body).Decode(dst)
	}

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
	UID         string
	CID         string
	ActiveRoute string
	ActiveCase  model.Case
}

func GetEnv(store *model.Store, r *http.Request) Env {
	kase := GetCase(store, r)
	user := GetUser(store, r)

	return Env{
		UID:         user.ID,
		CID:         kase.ID,
		ActiveRoute: r.URL.Path,
		ActiveCase:  kase,
	}
}

func GetUser(store *model.Store, r *http.Request) model.User {
	sess, _ := SessionStore.Get(r, SessionName)
	claims, _ := sess.Values["oidcClaims"].(map[string]interface{})
	uid, _ := claims[cmp.Or(os.Getenv("OIDC_ID_CLAIM"), "sub")].(string)
	user, _ := store.GetUser(uid)
	return user
}

func GetCase(store *model.Store, r *http.Request) model.Case {
	cid := r.PathValue("cid")
	kase, _ := store.GetCase(cid)
	return kase
}

func ServeDir(prefix string, dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.StripPrefix("/web/", fs)
}

func ServeFile(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, name)
	})
}

func random(n int) string {
	// random string
	var src = rand.NewSource(time.Now().UnixNano())

	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
