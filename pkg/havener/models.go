package havener

type Config struct {
	Name     string             `yaml:"name"`
	Releases map[string]Release `yaml:"releases"`
}

type Release struct {
	ChartName      string      `yaml:"chart_name"`
	ChartNamespace string      `yaml:"chart_namespace"`
	ChartLocation  string      `yaml:"chart_location"`
	ChartVersion   int         `yaml:"chart_version"`
	Overrides      interface{} `yaml:"overrides"`
}
