package agent

// ExternalAPI is the information for the ex
type ExternalAPI struct {
	Name string
	ID   string
	URL  string

	Spec []byte
	Icon string
	//	Instances []anypoint.APIInstance
	Packaging string

	Version       string
	CatalogType   string
	Description   string
	Documentation []byte
}
