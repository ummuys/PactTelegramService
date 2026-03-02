package tgapi

import "context"

type tgapi interface {
	// CreateSession() SessionInfoCh
	// SubmitPassword(sessionID string, password string) error
	// DeleteSession(sessionID string, ctx context.Context) error
	// SendMessage(sessionID string, peer string, text string, ctx context.Context) (int64, error)
	// SubscribeMessages(sessionID string, ctx context.Context) (<-chan BroadcastMessage, error)
}

type SessionInfoCh struct {
	SessionID string
	QrChan    chan string
	ErrChan   chan error
	StateChan chan SessionState
}

type SessionManager interface {
	CreateSession() SessionInfoCh
	SubmitPassword(sessionID, password string) error
	Get()
	Delete()
}

type Session interface {
	SendMessage(ctx context.Context, peer, text string)
	SubscribeMessages(ctx context.Context) (<-chan BroadcastMessage, error)
	Close() error
}
