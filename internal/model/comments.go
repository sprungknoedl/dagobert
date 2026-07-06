package model

import "slices"

// CommentKinds are the sub-object route segments that can carry comments; the
// values double as the table names of the parent objects.
var CommentKinds = []string{"events", "assets", "indicators", "evidences", "malware", "tasks"}

type Comment struct {
	ID       string
	CaseID   string
	Kind     string
	ObjectID string
	Author   string // user UPN
	Time     Time
	Message  string
	// AuthorName is resolved from the users table at query time and never
	// stored; it falls back to the raw UPN in the view when the user is gone.
	AuthorName string `gorm:"->" json:"-" form:"-"`
}

func (store *Store) ListComments(cid string, kind string, oid string) ([]Comment, error) {
	list := []Comment{}
	tx := store.DB.
		Select("comments.*, users.name AS author_name").
		Joins("LEFT JOIN users ON users.upn = comments.author").
		Where("comments.case_id = ? AND comments.kind = ? AND comments.object_id = ?", cid, kind, oid).
		Order("comments.time asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetComment(cid string, id string) (Comment, error) {
	obj := Comment{}
	tx := store.DB.First(&obj, "id = ? AND case_id = ?", id, cid)
	return obj, tx.Error
}

func (store *Store) SaveComment(cid string, obj Comment) error {
	obj.CaseID = cid
	if err := store.assertCaseOwnership(&Comment{}, obj.ID, cid); err != nil {
		return err
	}
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteComment(cid string, id string) error {
	return store.DB.Delete(&Comment{}, "id = ? AND case_id = ?", id, cid).Error
}

// CountComments returns the number of comments per object of a case, keyed
// `kind + ":" + object_id`. Used by the list views for the count badges.
func (store *Store) CountComments(cid string) (map[string]int, error) {
	rows := []struct {
		Kind     string
		ObjectID string
		N        int
	}{}
	err := store.DB.Table("comments").
		Select("kind, object_id, count(*) AS n").
		Where("case_id = ?", cid).
		Group("kind, object_id").
		Scan(&rows).Error

	counts := map[string]int{}
	for _, row := range rows {
		counts[row.Kind+":"+row.ObjectID] = row.N
	}
	return counts, err
}

// HasCaseObject reports whether the parent object exists in the case. A kind
// outside CommentKinds is simply not found — the check also keeps the
// user-controlled value out of the SQL (kind names the table).
func (store *Store) HasCaseObject(cid string, kind string, oid string) (bool, error) {
	if !slices.Contains(CommentKinds, kind) {
		return false, nil
	}

	var n int64
	err := store.DB.Table(kind).Where("id = ? AND case_id = ?", oid, cid).Count(&n).Error
	return n > 0, err
}

// deleteObjectComments removes the comments attached to one parent object;
// called from the parent Delete* store methods.
func (store *Store) deleteObjectComments(cid string, kind string, oid string) error {
	return store.DB.Delete(&Comment{}, "case_id = ? AND kind = ? AND object_id = ?", cid, kind, oid).Error
}
