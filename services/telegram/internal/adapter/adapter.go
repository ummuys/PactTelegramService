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
	chanels := tsa.svc.CreateSession()

	for {
		select {
		// CTX SIGNAL
		case <-ctx.Done():
			return nil

		// QR READY
		case qr, ok := <-chanels.QrChan:
			if !ok {
				tsa.logger.Warn().Msg("qr chanel is closed")
				return status.Error(codes.Internal, errs.ErrInternalServer.Error())
			}
			tsa.sendCreateSessionResponse(stream, chanels.SessionID, qr)

		// ERR READY
		case err, ok := <-chanels.ErrChan:
			if !ok {
				tsa.logger.Warn().Msg("err chanel is closed")
				return status.Error(codes.Internal, errs.ErrInternalServer.Error())
			}

			switch {
			case err == nil:
				tsa.sendCreateSessionState(stream, tsv1.SessionStatus_SESSION_STATUS_AUTHORIZED, "auth successful")
				return nil
			default:
				tsa.logger.Warn().Err(err).Msg("")
				return status.Error(codes.Internal, errs.ErrInternalServer.Error())
			}

		// SESSION SIGNAL READY
		case state, ok := <-chanels.StateChan:
			if !ok {
				tsa.logger.Warn().Msg("status chanel is closed")
				return status.Error(codes.Internal, errs.ErrInternalServer.Error())
			}
			switch state {
			case tgapi.StateNeedPassword:
				tsa.sendCreateSessionState(stream, tsv1.SessionStatus_SESSION_STATUS_PASSWORD_NEEDED, "you need to use SubmitPassword, because you have 2FA")
				return nil
			}
		}

	}
}

func (tsa *TelegramServiceAdapter) SubmitPassword(ctx context.Context, in *tsv1.SubmitPasswordRequest) (*tsv1.SubmitPasswordEvent, error) {

	if err := tsa.tgAPI.SubmitPassword(in.SessionId, in.Password); err != nil {
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

		case errors.Is(err, context.Canceled):
			return &tsv1.SubmitPasswordEvent{Event: &tsv1.SubmitPasswordEvent_SessionState{
				SessionState: &tsv1.SessionState{
					Status:  tsv1.SessionStatus_SESSION_STATUS_DEADLINE_EXCEEDED,
					Message: "session is dead, read qr again",
				},
			}}, nil

		default:
			tsa.logger.Error().Err(err).Msg("")
			return &tsv1.SubmitPasswordEvent{Event: &tsv1.SubmitPasswordEvent_Empty{}}, status.Error(codes.Internal, errs.ErrInternalServer.Error())
		}
	}
	return &tsv1.SubmitPasswordEvent{Event: &tsv1.SubmitPasswordEvent_SessionState{
		SessionState: &tsv1.SessionState{
			Status:  tsv1.SessionStatus_SESSION_STATUS_AUTHORIZED,
			Message: "auth succsessful",
		},
	}}, nil
}

func (tsa *TelegramServiceAdapter) DeleteSession(ctx context.Context, in *tsv1.DeleteSessionRequest) (*emptypb.Empty, error) {
	if err := tsa.tgAPI.DeleteSession(in.SessionId, ctx); err != nil {
		switch {
		case errors.Is(err, errs.ErrSessionNotFound):
			tsa.logger.Error().Err(err).Msg("")
			return &emptypb.Empty{}, status.Error(codes.NotFound, errs.ErrSessionNotFound.Error())
		default:
			tsa.logger.Error().Err(err).Msg("")
			return &emptypb.Empty{}, status.Error(codes.Internal, errs.ErrInternalServer.Error())
		}
	}
	return &emptypb.Empty{}, nil
}

func (tsa *TelegramServiceAdapter) SendMessage(ctx context.Context, in *tsv1.SendMessageRequest) (*tsv1.SendMessageResponse, error) {
	id, err := tsa.tgAPI.SendMessage(in.SessionId, in.Peer, in.Text, ctx)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrSessionNotFound):
			tsa.logger.Error().Err(err).Msg("")
			return &tsv1.SendMessageResponse{}, status.Error(codes.NotFound, errs.ErrSessionNotFound.Error())
		default:
			tsa.logger.Error().Err(err).Msg("")
			return &tsv1.SendMessageResponse{}, status.Error(codes.Internal, errs.ErrInternalServer.Error())
		}
	}
	return &tsv1.SendMessageResponse{MessageId: id}, nil
}

func (tsa *TelegramServiceAdapter) SubscribeMessages(in *tsv1.SubscribeMessagesRequest, stream tsv1.TelegramService_SubscribeMessagesServer) error {
	ctx := stream.Context()
	ch, err := tsa.tgAPI.SubscribeMessages(in.SessionId, ctx)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrSessionNotFound):
			tsa.logger.Error().Err(err).Msg("")
			return status.Error(codes.NotFound, errs.ErrSessionNotFound.Error())
		default:
			tsa.logger.Error().Err(err).Msg("")
			return status.Error(codes.Internal, errs.ErrInternalServer.Error())
		}
	}

	for {
		select {
		case msg := <-ch:
			stream.Send(&tsv1.MessageUpdate{
				MessageId: msg.MessageID,
				Text:      msg.Text,
				From:      msg.From,
				Timestamp: timestamppb.New(msg.Timestamp),
			})
		case <-ctx.Done():
			return nil
		}
	}
}
