package model

type Task struct {
	ID      string
	Type    string
	Task    string
	Done    bool
	Owner   string
	DateDue Time
	CaseID  string
	Custom  Custom `form:"-"`
}

func (store *Store) ListTasks(cid string) ([]Task, error) {
	list := []Task{}
	tx := store.DB.
		Where("case_id = ?", cid).
		Order("date_due asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetTask(cid string, id string) (Task, error) {
	obj := Task{}
	tx := store.DB.First(&obj, "id = ? AND case_id = ?", id, cid)
	return obj, tx.Error
}

func (store *Store) SaveTask(cid string, obj Task) error {
	obj.CaseID = cid
	if err := store.assertCaseOwnership(&Task{}, obj.ID, cid); err != nil {
		return err
	}
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteTask(cid string, id string) error {
	return store.DB.Delete(&Task{}, "id = ? AND case_id = ?", id, cid).Error
}
