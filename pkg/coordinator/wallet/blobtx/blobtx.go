package blobtx

import (
	"crypto/rand"
	"fmt"
	"io"
	mathRand "math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
)

func GenerateBlobSidecar(blobRefs []string, txIdx, replacementIdx uint64) ([]common.Hash, *types.BlobTxSidecar, error) {
	blobHashes := make([]common.Hash, 0)
	blobSidecar := &types.BlobTxSidecar{
		Blobs:       make([]kzg4844.Blob, 0),
		Commitments: make([]kzg4844.Commitment, 0),
		Proofs:      make([]kzg4844.Proof, 0),
	}

	for idx, blobRef := range blobRefs {
		blobCommitment, err := parseBlobRefs(blobRef, txIdx, idx, replacementIdx)
		if err != nil {
			return nil, nil, err
		}

		blobHashes = append(blobHashes, blobCommitment.VersionedHash)
		blobSidecar.Blobs = append(blobSidecar.Blobs, blobCommitment.Blob)
		blobSidecar.Commitments = append(blobSidecar.Commitments, blobCommitment.Commitment)
		blobSidecar.Proofs = append(blobSidecar.Proofs, blobCommitment.Proof)
	}

	return blobHashes, blobSidecar, nil
}

func parseBlobRefs(blobRefs string, txIdx uint64, blobIdx int, replacementIdx uint64) (*BlobCommitment, error) {
	var err error

	var blobBytes []byte

	for _, blobRef := range strings.Split(blobRefs, ",") {
		var blobRefBytes []byte
		if strings.HasPrefix(blobRef, "0x") {
			blobRefBytes = common.FromHex(blobRef)
		} else {
			refParts := strings.Split(blobRef, ":")
			switch refParts[0] {
			case "identifier":
				// just some identifier to make assertoor blobs more recognizable
				blobLabel := fmt.Sprintf("0x1611BB0000%08dFF%02dFF%04dFEED", txIdx, blobIdx, replacementIdx)
				blobRefBytes = common.FromHex(blobLabel)
			case "file":
				// load blob data from local file
				blobRefBytes, err = os.ReadFile(strings.Join(refParts[1:], ":"))
				if err != nil {
					return nil, err
				}
			case "url":
				// load blob data from remote url
				blobRefBytes, err = loadURLRef(strings.Join(refParts[1:], ":"))
				if err != nil {
					return nil, err
				}
			case "repeat":
				// repeat hex string
				if len(refParts) != 3 {
					return nil, fmt.Errorf("invalid repeat ref format: %v", blobRef)
				}
				repeatCount, err2 := strconv.Atoi(refParts[2])
				if err2 != nil {
					return nil, fmt.Errorf("invalid repeat count: %v", refParts[2])
				}
				repeatBytes := common.FromHex(refParts[1])
				repeatBytesLen := len(repeatBytes)
				blobRefBytes = make([]byte, repeatCount*repeatBytesLen)
				for i := 0; i < repeatCount; i++ {
					copy(blobRefBytes[(i*repeatBytesLen):], repeatBytes)
				}
			case "random":
				// random blob data
				var blobLen int

				if len(refParts) > 1 {
					var err2 error

					blobLen, err2 = strconv.Atoi(refParts[2])
					if err2 != nil {
						return nil, fmt.Errorf("invalid repeat count: %v", refParts[2])
					}
				} else {
					//nolint:gosec // ignore
					blobLen = mathRand.Intn((params.BlobTxFieldElementsPerBlob * (params.BlobTxBytesPerFieldElement - 1)) - len(blobBytes))
				}
				blobRefBytes, err = randomBlobData(blobLen)
				if err != nil {
					return nil, err
				}
			}
		}

		if blobRefBytes == nil {
			return nil, fmt.Errorf("unknown blob ref: %v", blobRef)
		}

		blobBytes = append(blobBytes, blobRefBytes...)
	}

	blobCommitment, err := EncodeBlob(blobBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid blob: %w", err)
	}

	return blobCommitment, nil
}

func loadURLRef(url string) ([]byte, error) {
	//nolint:gosec // ignore
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("received http error: %v", response.Status)
	}

	return io.ReadAll(response.Body)
}

func randomBlobData(size int) ([]byte, error) {
	data := make([]byte, size)

	n, err := rand.Read(data)
	if err != nil {
		return nil, err
	}

	if n != size {
		return nil, fmt.Errorf("could not create random blob data with size %d: %v", size, err)
	}

	return data, nil
}
