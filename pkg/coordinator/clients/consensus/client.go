package consensus

import (
	"context"

	"github.com/sirupsen/logrus"
)

func GetConsensusClient(ctx context.Context, log logrus.FieldLogger, url string) *Client {
	client := NewConsensusClient(log, url)

	return &client
}
