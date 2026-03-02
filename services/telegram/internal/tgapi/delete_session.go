package tgapi

import (
	"context"

	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
)

func (c *tgCli) DeleteSession(sessionID string, ctx context.Context) error {
	c.authManager.closeAuthSession(sessionID)

	data, ok := c.repos.Get(sessionID)
	if !ok {
		return errs.ErrSessionNotFound
	}

	cli := c.newClientWithStorage(data, nil, ctx)

	if err := cli.Run(ctx, func(rctx context.Context) error {
		_, err := cli.API().AuthLogOut(rctx)
		return err
	}); err != nil {
		return err
	}

	c.repos.Delete(sessionID)
	return nil
}
