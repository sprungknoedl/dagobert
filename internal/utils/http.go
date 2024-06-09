package utils

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

var decoder = schema.NewDecoder()
var SessionName = "default"
var SessionStore = sessions.NewCookieStore([]byte(os.Getenv("WEB_SESSION_SECRET")))

func Warn(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	log.Printf("| %s | %v", tty.Yellow("WAR"), err)

	w.Header().Add("HX-Retarget", "#errors")
	w.Header().Add("HX-Reswap", "beforeend")
	render(w, r, "internal/views/toasts-warning.html", map[string]any{"err": err})
}

func Err(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	log.Printf("| %s | %v", tty.Red("ERR"), err)

	w.Header().Add("HX-Retarget", "#errors")
	w.Header().Add("HX-Reswap", "beforeend")
	render(w, r, "internal/views/toasts-error.html", map[string]any{"err": err})
}

func Render(store *model.Store, w http.ResponseWriter, r *http.Request, name string, values map[string]any) {
	values["env"] = GetEnv(store, r)
	values["model"] = map[string]any{
		"AssetTypes":     model.AssetTypes,
		"CaseSeverities": model.CaseSeverities,
		"CaseOutcomes":   model.CaseOutcomes,
		"EventTypes":     model.EventTypes,
		"EvidenceTypes":  model.EvidenceTypes,
		"IndicatorTypes": model.IndicatorTypes,
		"IndicatorTLPs":  model.IndicatorTLPs,
		"TaskTypes":      model.TaskTypes,
	}

	render(w, r, name, values)
}

func render(w http.ResponseWriter, r *http.Request, name string, values map[string]any) {
	tpl, err := template.ParseFiles(
		name,
		"internal/views/_layout.html",
		"internal/views/_icons.html",
	)
	if err != nil {
		log.Printf("| %s | %v", tty.Red("ERR"), err)
		return
	}

	if err = tpl.Execute(w, values); err != nil {
		Err(w, r, err)
	}
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
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
		ActiveRoute: r.RequestURI,
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
