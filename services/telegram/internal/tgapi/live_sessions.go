package tgapi

import (
	"context"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/rs/zerolog"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
)

type LiveSession struct {
	sessionID string

	appID   int
	appHash string

	logger zerolog.Logger

	storage *session.StorageMemory

	cmdCh   chan commands     // канал для задач
	qrCh    chan string       // канал для получения qrCode
	stateCh chan SessionState // канал для получения состояния клиента
	errCh   chan error        // канал для получения ошибок
	cmdCh   chan commands     // канал для получения команд после авторизации
	passCh  chan passReq      // канал для получения пароля

	cancel context.CancelFunc // закончить сессию

}

type passReq struct {
	password string
	resp     chan error
}

func (ls *LiveSession) Run(ctx context.Context) {

	dispatcher := tg.NewUpdateDispatcher()
	cli := telegram.NewClient(ls.appID, ls.appHash, telegram.Options{
		UpdateHandler:  dispatcher,
		SessionStorage: ls.storage,
	})

	_ = cli.Run(ctx, func(rctx context.Context) error {
		err := ls.stepGetQr(ctx, cli, dispatcher)
		if err != nil {

		}

		return nil
	})

}

func (ls *LiveSession) stepGetQr(ctx context.Context, cli *telegram.Client, dispatcher tg.UpdateDispatcher) error {
	tries := 0
	attempt := 10
	accept := qrlogin.OnLoginToken(dispatcher)
	show := func(ctx context.Context, token qrlogin.Token) error {
		if tries >= attempt {
			return errs.ErrQrTimeout
		}
		tries++
		ls.qrCh <- token.URL()
		return nil
	}
	_, err := cli.QR().Auth(ctx, accept, show)
	return err
}

func (ls *LiveSession) stepGetPasswordFor2FA(ctx context.Context, cli *telegram.Client) error {
	for {

		msg, ok := <-ls.passCh
		if !ok {
			return nil
		}

		_, err := cli.Auth().Password(ctx, msg.password)
		if err != nil {
			msg.resp <- err
			continue
		}

		msg.resp <- nil
		return nil

	}
}
