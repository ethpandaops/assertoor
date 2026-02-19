package buildinfo

import "fmt"

var BuildVersion string
var BuildRelease string

func GetVersion() string {
	if BuildVersion == "" {
		return "local build"
	}

	if BuildRelease == "" {
		return fmt.Sprintf("git-%v", BuildVersion)
	}

	return fmt.Sprintf("%v (git-%v)", BuildRelease, BuildVersion)
}
