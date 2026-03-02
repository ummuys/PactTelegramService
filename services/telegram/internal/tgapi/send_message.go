package tgapi

import (
	"context"

	"github.com/gotd/td/telegram/message"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
)

func (c *tgCli) SendMessage(sessionID string, peer string, text string, ctx context.Context) (int64, error) {

	data, ok := c.repos.Get(sessionID)
	if !ok {
		return 0, errs.ErrSessionNotFound
	}

	cli := c.newClientWithStorage(data, nil, ctx)

	err := cli.Run(ctx, func(rctx context.Context) error {
		s := message.NewSender(cli.API())
		_, err := s.Resolve(peer).Text(rctx, text)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return 100, nil

}
