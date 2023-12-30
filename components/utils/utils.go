package utils

type CaseDTO struct {
	ID   int64
	Name string
}

type Env struct {
	Routes      func(name string, params ...interface{}) string
	Username    string
	ActiveRoute string
	ActiveCase  CaseDTO
}
