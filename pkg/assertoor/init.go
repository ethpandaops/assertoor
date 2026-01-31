package assertoor

import (
	hbls "github.com/herumi/bls-eth-go-binary/bls"
)

func init() {
	//nolint:errcheck // ignore
	hbls.Init(hbls.BLS12_381)
	//nolint:errcheck // ignore
	hbls.SetETHmode(hbls.EthModeLatest)
}
