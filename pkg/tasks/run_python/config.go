package runpython

import "errors"

type Config struct {
	PythonPath   string            `yaml:"pythonPath" json:"pythonPath" desc:"Python interpreter to use when no uv venv is available. Default: 'python3' from PATH."`
	PythonArgs   []string          `yaml:"pythonArgs" json:"pythonArgs" desc:"Additional arguments to pass to python before the script path."`
	EnvVars      map[string]string `yaml:"envVars" json:"envVars" format:"expressionMap" desc:"Variables to expose to the script. Each value is a runtime variable query resolved at task start. JSON-encoded into env vars and also surfaced as the 'env' dict inside the script."`
	Script       string            `yaml:"script" json:"script" require:"A" format:"python" desc:"The Python source to execute."`
	UseUV        bool              `yaml:"useUv" json:"useUv" desc:"If true, run the script inside the uv-managed venv referenced by 'venvVar'. When the variable is unset, the task auto-initializes a default venv and registers a cleanup task. Default: true."`
	VenvVar      string            `yaml:"venvVar" json:"venvVar" desc:"Variable name to read the uv venv path from (and to populate during auto-init). Default: 'python_uv_path'."`
	UVPath       string            `yaml:"uvPath" json:"uvPath" desc:"Path to the uv binary used for auto-init. Default: 'uv' from PATH."`
	Requirements []string          `yaml:"requirements" json:"requirements" desc:"Pip-installable specs to ensure are present in the venv before running. Installed via 'uv pip install --python <venv>/bin/python'."`
}

func DefaultConfig() Config {
	return Config{
		PythonPath: "python3",
		PythonArgs: []string{},
		UseUV:      true,
		VenvVar:    "python_uv_path",
		UVPath:     "uv",
	}
}

func (c *Config) Validate() error {
	if c.Script == "" {
		return errors.New("script must be specified")
	}

	if c.PythonPath == "" {
		c.PythonPath = "python3"
	}

	if c.VenvVar == "" {
		c.VenvVar = "python_uv_path"
	}

	if c.UVPath == "" {
		c.UVPath = "uv"
	}

	return nil
}
