package adapter

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	tsv1 "github.com/ummuys/pacttelegramservice/api/pb/v1"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/service"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/tgapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TelegramServiceAdapter struct {
	tsv1.UnimplementedTelegramServiceServer
	logger zerolog.Logger
	svc    service.TelegramService
}

func NewTelegramServiceAdapter(svc service.TelegramService, baseLogger zerolog.Logger) *TelegramServiceAdapter {
	logger := baseLogger.With().Str("component", "adapter").Logger()
	return &TelegramServiceAdapter{svc: svc, logger: logger}
}

func (tsa *TelegramServiceAdapter) CreateSession(_ *emptypb.Empty, stream tsv1.TelegramService_CreateSessionServer) error {
	ctx := stream.Context()
	ch := tsa.svc.CreateSession(ctx)

	tsa.logger.Info().Str("session_id", ch.SessionID).Msg("create session stream started")
	defer tsa.logger.Info().Str("session_id", ch.SessionID).Msg("create session stream finished")

	cleanup := func() {
		_ = tsa.svc.DeleteSession(ch.SessionID)
	}

	fail := func(level string, err error, msg string, grpcErr error) error {
		tsa.logErr(level, err, msg, func(e *zerolog.Event) *zerolog.Event {
			return e.Str("session_id", ch.SessionID)
		})
		cleanup()
		return grpcErr
	}

	for {
		select {
		case <-ctx.Done():
			cleanup()
			return status.FromContextError(ctx.Err()).Err()

		case qr, ok := <-ch.QrChan:
			if !ok {
				if ctx.Err() != nil {
					cleanup()
					return status.FromContextError(ctx.Err()).Err()
				}
				cleanup()
				return status.Error(codes.Internal, errs.ErrInternalServer.Error())
			}
			if err := tsa.sendCreateSessionResponse(stream, ch.SessionID, qr); err != nil {
				cleanup()
				return fail("warn", err, "stream send qr failed", grpcStatusFromError(err))
			}

		case state, ok := <-ch.StateChan:
			if !ok {
				if ctx.Err() != nil {
					cleanup()
					return status.FromContextError(ctx.Err()).Err()
				}
				cleanup()
				return status.Error(codes.Internal, errs.ErrInternalServer.Error())
			}

			switch state {
			case tgapi.StateNeedPassword:
				if err := tsa.sendCreateSessionState(stream,
					tsv1.SessionStatus_SESSION_STATUS_PASSWORD_NEEDED,
					"2FA enabled: call SubmitPassword with session_id and password",
				); err != nil {
					return fail("warn", err, "stream send state failed", grpcStatusFromError(err))
				}
				return nil

			case tgapi.StateAuthSuccessful:
				if err := tsa.sendCreateSessionState(stream,
					tsv1.SessionStatus_SESSION_STATUS_AUTHORIZED,
					"authorized",
				); err != nil {
					return fail("warn", err, "stream send state failed", grpcStatusFromError(err))
				}
				return nil

			default:
				cleanup()
				return status.Error(codes.Internal, errs.ErrUnknownClientEvent.Error())
			}

		case err, ok := <-ch.ErrChan:
			if !ok {
				if ctx.Err() != nil {
					cleanup()
					return status.FromContextError(ctx.Err()).Err()
				}
				cleanup()
				return status.Error(codes.Internal, errs.ErrInternalServer.Error())
			}
			return fail("warn", err, "create session error", grpcStatusFromError(err))
		}
	}
}

func (tsa *TelegramServiceAdapter) SubmitPassword(ctx context.Context, in *tsv1.SubmitPasswordRequest) (*tsv1.SubmitPasswordEvent, error) {
	err := tsa.svc.SubmitPassword(in.SessionId, in.Password)
	if err == nil {
		return &tsv1.SubmitPasswordEvent{
			Event: &tsv1.SubmitPasswordEvent_SessionState{
				SessionState: &tsv1.SessionState{
					Status:  tsv1.SessionStatus_SESSION_STATUS_AUTHORIZED,
					Message: "authorized",
				},
			},
		}, nil
	}

	tsa.logErr("warn", err, "submit password failed",
		func(e *zerolog.Event) *zerolog.Event { return e.Str("session_id", in.SessionId) },
	)

	switch {
	case errors.Is(err, errs.ErrInvalidPassword):
		return &tsv1.SubmitPasswordEvent{Event: &tsv1.SubmitPasswordEvent_SessionState{
			SessionState: &tsv1.SessionState{
				Status:  tsv1.SessionStatus_SESSION_STATUS_UNAUTHENTICATED,
				Message: err.Error(),
			},
		}}, nil

	case errors.Is(err, errs.ErrSessionNotFound):
		return &tsv1.SubmitPasswordEvent{Event: &tsv1.SubmitPasswordEvent_SessionState{
			SessionState: &tsv1.SessionState{
				Status:  tsv1.SessionStatus_SESSION_STATUS_SESSION_NOT_FOUND,
				Message: err.Error(),
			},
		}}, nil

	case errors.Is(err, errs.ErrQrTimeout), errors.Is(err, context.DeadlineExceeded):
		return &tsv1.SubmitPasswordEvent{Event: &tsv1.SubmitPasswordEvent_SessionState{
			SessionState: &tsv1.SessionState{
				Status:  tsv1.SessionStatus_SESSION_STATUS_DEADLINE_EXCEEDED,
				Message: "session expired, read QR again",
			},
		}}, nil

	case errors.Is(err, errs.ErrSessionClosed), errors.Is(err, context.Canceled):
		return &tsv1.SubmitPasswordEvent{Event: &tsv1.SubmitPasswordEvent_SessionState{
			SessionState: &tsv1.SessionState{
				Status:  tsv1.SessionStatus_SESSION_STATUS_UNAUTHENTICATED,
				Message: "session closed",
			},
		}}, nil

	default:

		return &tsv1.SubmitPasswordEvent{Event: &tsv1.SubmitPasswordEvent_Empty{}},
			status.Error(codes.Internal, errs.ErrInternalServer.Error())
	}
}

func (tsa *TelegramServiceAdapter) DeleteSession(ctx context.Context, in *tsv1.DeleteSessionRequest) (*emptypb.Empty, error) {
	if err := tsa.svc.DeleteSession(in.SessionId); err != nil {
		tsa.logErr("warn", err, "delete session failed",
			func(e *zerolog.Event) *zerolog.Event { return e.Str("session_id", in.SessionId) },
		)
		return nil, grpcStatusFromError(err)
	}
	return &emptypb.Empty{}, nil
}

func (tsa *TelegramServiceAdapter) SendMessage(ctx context.Context, in *tsv1.SendMessageRequest) (*tsv1.SendMessageResponse, error) {
	id, err := tsa.svc.SendMessage(ctx, in.SessionId, in.Peer, in.Text)
	if err != nil {
		tsa.logErr("warn", err, "send message failed",
			func(e *zerolog.Event) *zerolog.Event { return e.Str("session_id", in.SessionId) },
		)
		return nil, grpcStatusFromError(err)
	}
	return &tsv1.SendMessageResponse{MessageId: id}, nil
}

func (tsa *TelegramServiceAdapter) SubscribeMessages(in *tsv1.SubscribeMessagesRequest, stream tsv1.TelegramService_SubscribeMessagesServer) error {
	ctx := stream.Context()

	ch, err := tsa.svc.SubscribeMessages(ctx, in.SessionId)
	if err != nil {
		tsa.logErr("warn", err, "subscribe messages failed",
			func(e *zerolog.Event) *zerolog.Event { return e.Str("session_id", in.SessionId) },
		)
		return grpcStatusFromError(err)
	}

	tsa.logger.Info().
		Str("session_id", in.SessionId).
		Msg("subscribe messages stream started")

	defer tsa.logger.Info().
		Str("session_id", in.SessionId).
		Msg("subscribe messages stream finished")

	for {
		select {
		case <-ctx.Done():
			return nil

		case msg, ok := <-ch:
			if !ok {
				return nil
			}

			if err := stream.Send(&tsv1.MessageUpdate{
				MessageId: msg.MessageID,
				Text:      msg.Text,
				From:      msg.From,
				Timestamp: timestamppb.New(msg.Timestamp),
			}); err != nil {
				tsa.logErr("warn", err, "stream send message update failed",
					func(e *zerolog.Event) *zerolog.Event { return e.Str("session_id", in.SessionId) },
				)
				return grpcStatusFromError(err)
			}
		}
	}
}
