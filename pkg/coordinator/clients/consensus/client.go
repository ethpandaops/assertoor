package consensus

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

func GetConsensusClient(ctx context.Context, log logrus.FieldLogger, url string) *Client {
	client := NewConsensusClient(log, url)
	if err := client.Bootstrap(ctx); err != nil {
		log.WithError(err).Error("failed to bootstrap consensus client")
	}

	for !client.Bootstrapped() {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second * 3):
			if err := client.Bootstrap(ctx); err != nil {
				log.WithError(err).Error("failed to bootstrap consensus client")
			}
		}
	}

	return &client
}
