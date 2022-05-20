package execution

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Client struct {
	log logrus.FieldLogger
}

func NewExecutionClient(log logrus.FieldLogger) Client {
	return Client{
		log: log,
	}
}

func (c *Client) Bootstrap(ctx context.Context) error {
	return nil
}
