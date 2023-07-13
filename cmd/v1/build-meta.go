package v1

type BuildMeta struct {
	BuildVersion string `json:"buildVersion" yaml:"buildVersion"`
	BuildDate    string `json:"buildDate" yaml:"buildDate"`
	BuildSHA     string `json:"buildSHA" yaml:"buildSHA"`

	IgnitionVersion string `json:"ignitionVersion" yaml:"ignitionVersion"`
	GolldpdVersion  string `json:"golldpdVersion" yaml:"golldpdVersion"`
	FrrVersion      string `json:"frrVersion" yaml:"frrVersion"`
}
