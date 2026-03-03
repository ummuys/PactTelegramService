package adapter

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	tsv1 "github.com/ummuys/pacttelegramservice/api/pb/v1"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (tsa *TelegramServiceAdapter) sendCreateSessionState(stream tsv1.TelegramService_CreateSessionServer, st tsv1.SessionStatus, msg string) error {
	err := stream.Send(&tsv1.CreateSessionEvent{
		Event: &tsv1.CreateSessionEvent_SessionState{
			SessionState: &tsv1.SessionState{
				Status:  st,
				Message: msg,
			},
		},
	})
	if err != nil {
		tsa.logger.Error().Err(err).Msg("can't send session state")
		return err
	}

	return nil
}

func (tsa *TelegramServiceAdapter) sendCreateSessionResponse(stream tsv1.TelegramService_CreateSessionServer, sessionID string, qrCode string) error {
	err := stream.Send(&tsv1.CreateSessionEvent{
		Event: &tsv1.CreateSessionEvent_SessionResponse{
			SessionResponse: &tsv1.CreateSessionResponse{
				SessionId: sessionID,
				QrCode:    qrCode,
			},
		},
	})
	if err != nil {
		tsa.logger.Error().Err(err).Msg("can't send session response")
		return err
	}

	return nil
}

func (tsa *TelegramServiceAdapter) logErr(level string, err error, msg string, fields ...func(e *zerolog.Event) *zerolog.Event) {
	ev := tsa.logger.Error()
	switch level {
	case "warn":
		ev = tsa.logger.Warn()
	case "info":
		ev = tsa.logger.Info()
	}
	ev = ev.Err(err)
	for _, f := range fields {
		ev = f(ev)
	}
	ev.Msg(msg)
}

func grpcStatusFromError(err error) error {
	if err == nil {
		return nil
	}

	switch {

	case errors.Is(err, errs.ErrSessionNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, errs.ErrSessionBroadcastClosed):
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, errs.ErrSessionClosed):

		return status.Error(codes.Canceled, err.Error())

	case errors.Is(err, errs.ErrInvalidPassword):
		return status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, errs.Err2FA):

		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, errs.ErrQrTimeout):
		return status.Error(codes.DeadlineExceeded, err.Error())

	case errors.Is(err, errs.ErrFloodWait):
		return status.Error(codes.ResourceExhausted, err.Error())

	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")

	default:
		return status.Error(codes.Internal, errs.ErrInternalServer.Error())
	}
}
