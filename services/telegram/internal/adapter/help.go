package adapter

import (
	tsv1 "github.com/ummuys/pacttelegramservice/api/pb/v1"
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
