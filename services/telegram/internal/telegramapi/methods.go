package telegramapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	gotdauth "github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/rs/zerolog"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/repository"
)

type tgCli struct {
	appID            int
	appHash          string
	appCtx           context.Context
	authManager      *authManager
	broadcastManager *broadcastManager
	logger           zerolog.Logger
	repos            repository.SessionRepository
}

const attempt = 10

func NewTelegramClient(sr repository.SessionRepository, appID int, appHash string, baseLogger zerolog.Logger, ctx context.Context) TelegramAPI {
	logger := baseLogger.With().Str("component", "tgcli").Logger()

	return &tgCli{repos: sr, appID: appID,
		appHash: appHash, appCtx: ctx,
		authManager:      newAuthManager(),
		broadcastManager: newBroadcastManager(),
		logger:           logger}
}

func (c *tgCli) CreateSession() SessionInfoCh {

	c.logger.Info().Msg("Catch new task: create client")

	storage := new(session.StorageMemory)
	dispatcher := tg.NewUpdateDispatcher()
	cli := telegram.NewClient(c.appID, c.appHash, telegram.Options{
		UpdateHandler:  dispatcher,
		SessionStorage: storage,
	})

	sessionID := uuid.New().String()

	qrCh := make(chan string, 10)
	errCh := make(chan error, 1)
	statusCh := make(chan SessionState, 1)

	go func() {

		defer close(qrCh)
		defer close(errCh)
		defer close(statusCh)

		authCtx, cancel := context.WithTimeout(c.appCtx, time.Second*60)
		c.authManager.newAuthSession(sessionID, authCtx, cancel)

		defer c.authManager.closeAuthSession(sessionID)
		defer cancel()

		tries := 0

		runErr := cli.Run(authCtx, func(rctx context.Context) error {
			accept := qrlogin.OnLoginToken(dispatcher)
			show := func(ctx context.Context, token qrlogin.Token) error {
				if tries >= attempt {
					return errs.ErrQrTimeout
				}

				tries++
				qrCh <- token.URL()
				return nil
			}

			_, err := cli.QR().Auth(rctx, accept, show)
			if err != nil {

				// 2FA
				if isPasswordNeeded(err) {
					statusCh <- StateNeedPassword

					for {

						password, replyCh, cerr := c.authManager.waitPassword(sessionID)
						if cerr != nil {
							return cerr
						}

						// try auth
						_, perr := cli.Auth().Password(rctx, password)
						if perr != nil {

							c.logger.Warn().Err(perr).Msg("cli.Auth().Password()")
							replyCh <- errs.ErrInvalidPassword
							continue

							// TO FIX !!!

							// fmt.Println("down here")
							// replyCh <- perr
							// return perr
						}

						replyCh <- nil
						return nil

					}

				}
				return err

			}
			return nil
		})

		if runErr == nil {
			data, derr := storage.LoadSession(context.Background())
			if derr == nil {
				c.repos.Set(sessionID, data)
			}
			c.authManager.closeAuthSession(sessionID)
		}

		errCh <- runErr
	}()

	return SessionInfoCh{SessionID: sessionID, QrChan: qrCh, ErrChan: errCh, StateChan: statusCh}
}

func (c *tgCli) SubmitPassword(sessionID string, password string) error {
	return c.authManager.sendPassword(sessionID, password)

}

func (c *tgCli) DeleteSession(sessionID string, ctx context.Context) error {
	c.authManager.closeAuthSession(sessionID)

	data, ok := c.repos.Get(sessionID)
	if !ok {
		return errs.ErrSessionNotFound
	}

	cli := c.newClientWithStorage(data, tg.UpdateDispatcher{}, ctx)

	if err := cli.Run(ctx, func(rctx context.Context) error {
		_, err := cli.API().AuthLogOut(rctx)
		return err
	}); err != nil {
		return err
	}

	c.repos.Delete(sessionID)
	return nil
}

func (c *tgCli) SendMessage(sessionID string, peer string, text string, ctx context.Context) (int64, error) {

	data, ok := c.repos.Get(sessionID)
	if !ok {
		return 0, errs.ErrSessionNotFound
	}

	cli := c.newClientWithStorage(data, tg.UpdateDispatcher{}, ctx)

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

func (c *tgCli) SubscribeMessages(sessionID string, ctx context.Context) (<-chan BroadcastMessage, error) {

	data, have := c.repos.Get(sessionID)
	if !have {
		return nil, errs.ErrSessionNotFound
	}

	// 1) hub
	hub, ok := c.broadcastManager.GetHub(sessionID)
	if !ok {
		hub = c.broadcastManager.CreateHub(sessionID, nil)
	}

	// 2) подписчик
	listenerID := uuid.New().String()
	subCh, unsubscribe, isOpen := hub.Subscribe(listenerID)
	if !isOpen {
		return nil, errs.ErrSessionNotFound
	}

	// 3) cli + dispatcher
	dispatcher := tg.NewUpdateDispatcher()
	cli := c.newClientWithStorage(data, dispatcher, ctx)

	dispatcher.OnNewMessage(func(_ context.Context, _ tg.Entities, u *tg.UpdateNewMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return nil
		}
		if msg.Out || msg.Message == "" {
			return nil
		}

		hub.Broadcast(BroadcastMessage{
			MessageID: int64(msg.ID),
			Text:      msg.Message,
			From:      peerToString(msg.FromID),
			Timestamp: time.Unix(int64(msg.Date), 0),
		})
		return nil
	})

	// 5) Run
	go func() {
		defer unsubscribe()

		_ = cli.Run(ctx, func(rctx context.Context) error {
			<-rctx.Done()
			return rctx.Err()
		})
	}()

	return subCh, nil
}

// TO WORK
func isPasswordNeeded(err error) bool {
	return tgerr.Is(err, "SESSION_PASSWORD_NEEDED") ||
		(tgerr.IsCode(err, 400, 401) && tgerr.Is(err, "SESSION_PASSWORD_NEEDED"))
}

func isPasswordIncorrect(err error) bool {
	return errors.Is(err, gotdauth.ErrPasswordInvalid) ||
		tg.IsPasswordHashInvalid(err) ||
		tgerr.Is(err, "PASSWORD_HASH_INVALID")
}

func (c *tgCli) newClientWithStorage(data []byte, dispatcher tg.UpdateDispatcher, ctx context.Context) *telegram.Client {
	storage := new(session.StorageMemory)
	storage.StoreSession(ctx, data)

	cli := telegram.NewClient(c.appID, c.appHash, telegram.Options{
		SessionStorage: storage,
		UpdateHandler:  dispatcher,
	})

	return cli
}

func peerToString(p tg.PeerClass) string {
	if p == nil {
		return "unknown"
	}
	switch v := p.(type) {
	case *tg.PeerUser:
		return fmt.Sprintf("user:%d", v.UserID)
	case *tg.PeerChat:
		return fmt.Sprintf("chat:%d", v.ChatID)
	case *tg.PeerChannel:
		return fmt.Sprintf("channel:%d", v.ChannelID)
	default:
		return "unknown"
	}
}
