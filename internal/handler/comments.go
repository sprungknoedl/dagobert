package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

// commentParent validates the {kind} route segment and verifies the parent
// object exists in the case; every comment handler goes through it.
func (h *Handler) commentParent(w http.ResponseWriter, r *http.Request) (cid string, kind string, oid string, ok bool) {
	cid, kind, oid = r.PathValue("cid"), r.PathValue("kind"), r.PathValue("oid")
	found, err := h.Store.HasCaseObject(cid, kind, oid)
	if err != nil {
		Err(w, r, err)
		return "", "", "", false
	}
	if !found {
		http.NotFound(w, r)
		return "", "", "", false
	}

	return cid, kind, oid, true
}

// canModifyComment implements the author-or-admin guard for edits and deletes.
func canModifyComment(user model.User, obj model.Comment) bool {
	return obj.Author == user.UPN || user.Role == "Administrator"
}

func (h *Handler) CommentList(w http.ResponseWriter, r *http.Request) {
	cid, kind, oid, ok := h.commentParent(w, r)
	if !ok {
		return
	}

	list, err := h.Store.ListComments(cid, kind, oid)
	if err != nil {
		Err(w, r, err)
		return
	}

	// ?edit= prefills the form at the bottom with an existing comment
	edit := model.Comment{ID: "new"}
	if eid := r.URL.Query().Get("edit"); eid != "" {
		if edit, err = h.Store.GetComment(cid, eid); err != nil {
			Err(w, r, err)
			return
		}
		if !canModifyComment(GetUser(r), edit) {
			Forbidden(w, r)
			return
		}
	}

	Render(w, r, http.StatusOK, views.CommentsDialog(h.Env(r), kind, oid, list, edit, valid.ValidationError{}))
}

func (h *Handler) CommentSave(w http.ResponseWriter, r *http.Request) {
	cid, kind, oid, ok := h.commentParent(w, r)
	if !ok {
		return
	}

	id := r.PathValue("id")
	dto := model.Comment{}
	err := Decode(h.Store, r, &dto, ValidateComment)
	if vr, isVr := err.(valid.ValidationError); err != nil && isVr {
		list, lerr := h.Store.ListComments(cid, kind, oid)
		if lerr != nil {
			Err(w, r, lerr)
			return
		}
		dto.ID = id
		Render(w, r, http.StatusUnprocessableEntity, views.CommentsDialog(h.Env(r), kind, oid, list, dto, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// path values win over any form-posted fields
	user := GetUser(r)
	dto.ID, dto.CaseID, dto.Kind, dto.ObjectID = id, cid, kind, oid
	if id == "new" {
		dto.ID = fp.Random(10)
		dto.Author = user.UPN
		dto.Time = model.Time(time.Now())
	} else {
		old, err := h.Store.GetComment(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
		if !canModifyComment(user, old) {
			Forbidden(w, r)
			return
		}
		// edits overwrite the message in place; author and time are kept
		dto.Author, dto.Time = old.Author, old.Time
	}

	if err := h.Store.SaveComment(cid, dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/comments/%s/%s/", cid, kind, oid))
}

func (h *Handler) CommentDelete(w http.ResponseWriter, r *http.Request) {
	cid, kind, oid, ok := h.commentParent(w, r)
	if !ok {
		return
	}

	id := r.PathValue("id")
	obj, err := h.Store.GetComment(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}
	if !canModifyComment(GetUser(r), obj) {
		Forbidden(w, r)
		return
	}

	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/comments/%s/%s/%s?confirm=yes", cid, kind, oid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	if err := h.Store.DeleteComment(cid, id); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/comments/%s/%s/", cid, kind, oid), http.StatusSeeOther)
}
