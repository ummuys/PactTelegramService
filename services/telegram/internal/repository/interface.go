package repository

type SessionRepository interface {
	Set(sessionID string, session []byte)
	Get(sessionID string) ([]byte, bool)
	Delete(sessionID string)
}
