package utils

type CaseDTO struct {
	ID   string
	Name string
}

type Env struct {
	Routes      func(name string, params ...interface{}) string
	Username    string
	ActiveRoute string
	ActiveCase  CaseDTO
	Search      string
	Sort        string
}
