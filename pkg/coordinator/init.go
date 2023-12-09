package coordinator

import (
	hbls "github.com/herumi/bls-eth-go-binary/bls"
)

func init() {
	hbls.Init(hbls.BLS12_381)
	hbls.SetETHmode(hbls.EthModeLatest)
}
