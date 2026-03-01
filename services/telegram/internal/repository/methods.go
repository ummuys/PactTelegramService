package repository

import (
	"sync"
)

type sessionRepository struct {
	m   map[string][]byte
	mux sync.RWMutex
}

func NewSessionRepository() SessionRepository {
	m := make(map[string][]byte)
	return &sessionRepository{m: m}
}

func (sr *sessionRepository) Set(sessionID string, session []byte) {
	sr.mux.Lock()
	defer sr.mux.Unlock()
	sr.m[sessionID] = session
}

func (sr *sessionRepository) Get(sessionID string) ([]byte, bool) {
	sr.mux.RLock()
	defer sr.mux.RUnlock()
	s, ok := sr.m[sessionID]
	return s, ok
}

func (sr *sessionRepository) Delete(sessionID string) {
	sr.mux.Lock()
	defer sr.mux.Unlock()
	delete(sr.m, sessionID)
}
