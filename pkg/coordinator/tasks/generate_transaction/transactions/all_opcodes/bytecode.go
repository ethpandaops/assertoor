package allopcodes

import (
	"embed"

	"github.com/ethereum/go-ethereum/common"
)

var (
	//go:embed *.hex
	embedFS embed.FS
)

func GetBytecode(name string) ([]byte, error) {
	data, err := embedFS.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return common.FromHex(string(data)), nil
}
