package domain

type storeState struct {
	path     string
	exists   bool
	password string
	missing  []string
}

type StoreView struct {
	Path     string
	Exists   bool
	Mappings []Mapping
}
