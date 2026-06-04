package runpythonuv

import "errors"

type Config struct {
	UVPath        string   `yaml:"uvPath" json:"uvPath" desc:"Path to the uv binary. Default: 'uv' from PATH."`
	PythonVersion string   `yaml:"pythonVersion" json:"pythonVersion" desc:"Optional Python version pin (e.g. '3.12'). Passed to 'uv venv --python'. uv downloads the requested CPython if missing."`
	Requirements  []string `yaml:"requirements" json:"requirements" desc:"Pip-installable specs to install into the venv (e.g. ['web3', 'requests>=2.32'])."`
	VenvVar       string   `yaml:"venvVar" json:"venvVar" desc:"Name of the variable to populate with the venv path. Subsequent run_python tasks read this. Default: 'python_uv_path'."`
	SkipIfSet     bool     `yaml:"skipIfSet" json:"skipIfSet" desc:"If true, skip setup when 'venvVar' is already set. Defaults to true."`
}

func DefaultConfig() Config {
	return Config{
		UVPath:    "uv",
		VenvVar:   "python_uv_path",
		SkipIfSet: true,
	}
}

func (c *Config) Validate() error {
	if c.UVPath == "" {
		return errors.New("uvPath must be specified")
	}

	if c.VenvVar == "" {
		return errors.New("venvVar must be specified")
	}

	return nil
}
