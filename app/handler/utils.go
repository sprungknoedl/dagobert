package handler

import (
	"cmp"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

var ZeroID string = "0"
var ZeroTime time.Time

var decoder = schema.NewDecoder()
var SessionName = "default"
var SessionStore = sessions.NewCookieStore([]byte(os.Getenv("WEB_SESSION_SECRET")))

func ImportCSV(store *model.Store, acl *ACL, w http.ResponseWriter, r *http.Request, uri string, numFields int, cb func(rec []string)) error {
	if r.Method == http.MethodGet {
		views.ImportDialog().Render(r.Context(), w)
		return nil
	}

	fr, _, err := r.FormFile("file")
	if err != nil {
		Warn(w, r, err)
		return err
	}

	cr := csv.NewReader(fr)
	cr.FieldsPerRecord = numFields
	_, err = cr.Read()                                                          // skip header
	if perr, ok := err.(*csv.ParseError); ok && perr.Err == csv.ErrFieldCount { // try semicolon instead, Excel often exports CSVs with ;
		fr.Seek(0, 0)
		cr = csv.NewReader(fr)
		cr.Comma = ';'
		cr.FieldsPerRecord = numFields
		cr.Read() // skip header
	}

	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			Warn(w, r, err)
			return err
		}

		cb(rec)
	}

	http.Redirect(w, r, uri, http.StatusSeeOther)
	return nil
}

func Warn(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	slog.Warn("400: Bad Request",
		"err", err,
		"raddr", r.RemoteAddr,
		"method", r.Method,
		"url", r.URL)
	w.WriteHeader(http.StatusBadRequest)
	views.ToastWarning(err).Render(r.Context(), w)
}

func Err(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	slog.Error("500: Internal Server Error",
		"err", err,
		"raddr", r.RemoteAddr,
		"method", r.Method,
		"url", r.URL)
	w.WriteHeader(http.StatusInternalServerError)
	views.ToastError(err).Render(r.Context(), w)
}

func Serve4xx(w http.ResponseWriter, r *http.Request) {
	Warn(w, r, errors.New("400: Client Test Error"))
}

func Serve5xx(w http.ResponseWriter, r *http.Request) {
	Err(w, r, errors.New("500: Internal Test Error"))
}

func JoinV(errs ...error) error {
	verrs := fp.Apply(fp.Filter(errs,
		func(err error) bool { _, ok := err.(valid.ValidationError); return err != nil && ok }),
		func(in error) valid.ValidationError { return in.(valid.ValidationError) })
	other := fp.Filter(errs,
		func(err error) bool { _, ok := err.(valid.ValidationError); return err != nil && !ok })
	if len(other) > 0 {
		return errors.Join(other...)
	}

	vr := valid.ValidationError{}
	for _, m := range verrs {
		maps.Copy(vr, m)
	}

	if vr.Valid() {
		return nil
	} else {
		return vr
	}
}

func Decode[T any](db *model.Store, r *http.Request, dst T, validator func(T, model.Enums) valid.ValidationError) error {
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
	err := decoder.Decode(dst, r.PostForm)
	if merr, ok := err.(schema.MultiError); ok {
		vr := valid.ValidationError{}
		for key, val := range merr {
			cerr := val.(schema.ConversionError)
			vr[key] = valid.Condition{Name: key, Invalid: true, Message: cerr.Err.Error()}
		}
		return vr
	} else if err != nil {
		return err
	}

	if validator != nil {
		enums, err := db.ListEnums()
		if err != nil {
			return err
		}
		if vr := validator(dst, enums); !vr.Valid() {
			return vr
		}
	}

	return nil
}

type Ctrl interface {
	Store() *model.Store
	ACL() *ACL
}

type BaseCtrl struct {
	store *model.Store
	acl   *ACL
}

func (ctrl BaseCtrl) Store() *model.Store { return ctrl.store }
func (ctrl BaseCtrl) ACL() *ACL           { return ctrl.acl }

func Env(ctrl Ctrl, r *http.Request) views.Env {
	kase := GetCase(ctrl.Store(), r)
	user := GetUser(ctrl.Store(), r)
	enums, _ := ctrl.Store().ListEnums()

	return views.Env{
		Route: r.URL.Path,
		Case:  kase,
		User:  user,
		Enums: enums,
		Allowed: func(method, url string) (string, bool) {
			return url, ctrl.ACL().Allowed(user.ID, url, method)
		},
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

func GetObject[T any](id string, obj T, getfn func(string) (T, error)) (T, error) {
	if id != "new" {
		return getfn(id)
	}

	return obj, nil
}

func ServeDir(prefix string, root fs.FS) http.Handler {
	fs := http.FileServer(http.FS(root))
	return http.StripPrefix("/public/", fs)
}

func ServeFile(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, name)
	})
}

func Render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.WriteHeader(status)
	if err := c.Render(r.Context(), w); err != nil {
		slog.Error("failed to render template",
			"err", err,
			"raddr", r.RemoteAddr,
			"method", r.Method,
			"status", status,
			"url", r.URL)
	}
}
