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
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-playground/form/v4"
	"github.com/sprungknoedl/dagobert/internal/auth"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/attck"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func ImportCSV(store *model.Store, acl *auth.ACL, w http.ResponseWriter, r *http.Request, uri string, numFields int, cb func(rec []string)) error {
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

func ListModules[T any](h *Handler, w http.ResponseWriter, r *http.Request, fn func(cid string, oid string) (T, error)) {
	oid := r.PathValue("id")
	cid := r.PathValue("cid")
	obj, err := fn(cid, oid)
	if err != nil {
		Err(w, r, err)
		return
	}

	modules := modules.Supported(obj)
	list, err := h.Store.GetJobs(oid)
	if err != nil {
		Err(w, r, err)
		return
	}

	jobs := fp.ToMap(list, func(j model.Job) string { return j.Name })
	runs := fp.Apply(modules, func(m model.Module) views.Job {
		// jobs[m.Name()] is the zero Job when the module never ran for this
		// object; the view relies on Name for the title and the schedule form
		job := jobs[m.Name()]
		job.Name = m.Name()
		return views.Job{
			Module: m,
			Job:    job,
		}
	})

	slices.SortFunc(runs, func(a, b views.Job) int { return cmp.Compare(a.Name, b.Name) })
	Render(w, r, http.StatusOK, views.ModuleList(h.Env(r), runs), nil)
}

func ScheduleModule[T any](h *Handler, w http.ResponseWriter, r *http.Request, fn func(cid string, oid string) (T, error)) {
	oid := r.PathValue("id")
	cid := r.PathValue("cid")
	obj, err := fn(cid, oid)
	if err != nil {
		Err(w, r, err)
		return
	}

	dto := model.Job{}
	err = Decode(h.Store, r, &dto, nil)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase, err := h.Store.GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	err = h.Store.PushJob(model.Job{
		ID:       fp.Random(10),
		Name:     dto.Name,
		Status:   "Scheduled",
		Case:     kase,
		ObjectID: oid,
		Object:   model.Object{Payload: obj},
		Settings: dto.Settings,
	})
	if err != nil {
		Err(w, r, err)
		return
	}

	ListModules(h, w, r, fn)
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
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
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
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	views.ToastError(err).Render(r.Context(), w)
}

func Forbidden(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(http.StatusText(http.StatusForbidden)))
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

func Decode[T any](db *model.Store, r *http.Request, dst T, validator func(T, model.ValueLists) valid.ValidationError) error {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
			return err
		}
		return runValidator(db, dst, validator)
	}

	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			return err
		}
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	decoder := form.NewDecoder()
	decoder.RegisterCustomTypeFunc(func(vals []string) (interface{}, error) {
		if len(vals) == 0 || vals[0] == "" {
			return model.Time{}, nil
		}
		var t model.Time
		if err := t.UnmarshalText([]byte(vals[0])); err != nil {
			return nil, err
		}
		return t, nil
	}, model.Time{})
	decoder.RegisterCustomTypeFunc(func(vals []string) (interface{}, error) {
		if len(vals) == 0 || vals[0] == "" {
			return model.Date{}, nil
		}
		var d model.Date
		if err := d.UnmarshalText([]byte(vals[0])); err != nil {
			return nil, err
		}
		return d, nil
	}, model.Date{})
	if err := decoder.Decode(dst, r.PostForm); err != nil {
		return err
	}

	return runValidator(db, dst, validator)
}

func runValidator[T any](db *model.Store, dst T, validator func(T, model.ValueLists) valid.ValidationError) error {
	if validator == nil {
		return nil
	}
	valueLists, err := db.ListValueLists()
	if err != nil {
		return err
	}
	if vr := validator(dst, valueLists); !vr.Valid() {
		return vr
	}
	return nil
}

// Handler holds the process-wide dependencies shared by all HTTP handlers.
// All handler methods hang off this one struct; routes are registered in init.go.
type Handler struct {
	Store      *model.Store
	ACL        *auth.ACL
	Mitre      *attck.KB
	Timesketch *timesketch.Client
}

func (h *Handler) Env(r *http.Request) views.Env {
	kase := GetCase(h.Store, r)
	user := GetUser(r)
	valueLists, _ := h.Store.ListValueLists()
	custom, _ := h.Store.ListCustomAttributes()

	return views.Env{
		Route:            r.URL.Path,
		Case:             kase,
		User:             user,
		ValueLists:       valueLists,
		CustomAttributes: custom,
		Allowed: func(method, url string) (string, bool) {
			return url, h.ACL.Allowed(user.ID, url, method)
		},
	}
}

// CollectCustom assembles the custom-attribute map from the form's cattr_*
// inputs after Decode has run. The Custom model field is tagged form:"-", so a
// broken custom section can never fail the core decode/save. Empty values are
// dropped (empty = delete the key); serialization is handled by Custom.Value.
func CollectCustom(r *http.Request) model.Custom {
	custom := model.Custom{}
	for key, vals := range r.PostForm {
		label, ok := strings.CutPrefix(key, "cattr_")
		if !ok || len(vals) == 0 || vals[0] == "" {
			continue
		}
		custom[label] = vals[0]
	}
	return custom
}

func GetUser(r *http.Request) model.User {
	user, err := auth.CurrentUser(r)
	if err != nil {
		slog.Error("failed to get current user", "err", err)
		return model.User{}
	}

	return *user
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
	return http.StripPrefix(prefix, fs)
}

// wantsJSON reports whether the response should be JSON rather than HTML.
// ApiKeyMiddleware defaults the Accept header for keyed clients that sent none,
// so this is the single predicate the whole response path consults.
func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

// Render renders c for browser clients. When the client wants JSON and data is
// non-nil, it marshals data instead — c is a lazy templ closure, so building an
// unused component costs nothing.
func Render(w http.ResponseWriter, r *http.Request, status int, c templ.Component, data any) {
	if wantsJSON(r) && data != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(data)
		return
	}

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

// RedirectAfterSave issues the post-save redirect to a list page for browser
// (Unpoly) clients, but answers JSON clients with 201 Created and the saved
// record. API clients follow 3xx redirects automatically; bouncing them to an
// HTML list page they may not be permitted to read (e.g. a create-only Donald
// key) would turn a successful write into a spurious 403.
func RedirectAfterSave(w http.ResponseWriter, r *http.Request, url string, record any) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(record)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}
