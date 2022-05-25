package execution

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

func GetExecutionClient(ctx context.Context, log logrus.FieldLogger, url string) *Client {
	client := NewExecutionClient(log, url)
	if err := client.Bootstrap(ctx); err != nil {
		log.WithError(err).Error("failed to bootstrap execution client")
	}

	for !client.Bootstrapped() {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second * 3):
			if err := client.Bootstrap(ctx); err != nil {
				log.WithError(err).Error("failed to bootstrap execution client")
			}
		}
	}

	return &client
}
