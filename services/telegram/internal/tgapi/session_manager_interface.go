package tgapi

type SessionInfoCh struct {
	SessionID string
	QrChan    chan string
	ErrChan   chan error
	StateChan chan SessionState
}

type SessionManager interface {
	CreateSession() SessionInfoCh
	SubmitPassword(sessionID, password string) error
	Get(sessionID string) Session
	Delete(sessionID string)
}
