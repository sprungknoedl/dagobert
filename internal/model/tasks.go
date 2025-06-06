package model

type Task struct {
	ID      string
	Type    string
	Task    string
	Done    bool
	Owner   string
	DateDue Time
	CaseID  string
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
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveTask(cid string, obj Task) error {
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteTask(cid string, id string) error {
	return store.DB.Delete(&Task{}, "id = ?", id).Error
}
