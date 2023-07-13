package v1

type BuildMeta struct {
	BuildVersion string `json:"buildVersion"`
	BuildDate    string `json:"buildDate"`
	BuildSHA     string `json:"buildSHA"`

	IgnitionVersion string `json:"ignitionVersion"`
	GolldpdVersion  string `json:"golldpdVersion"`
	FrrVersion      string `json:"frrVersion"`
}
