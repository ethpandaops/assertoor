package txpool

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func CreateDummyTransaction(nonce uint64, chainID *big.Int, privateKey *ecdsa.PrivateKey) (*types.Transaction, error) {
	// create a dummy transaction, we don't care about the actual data
	toAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    big.NewInt(100),
		Gas:      21000,
		GasPrice: big.NewInt(1),
		Data: 		nil,
	})

	signer := types.LatestSignerForChainID(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}
