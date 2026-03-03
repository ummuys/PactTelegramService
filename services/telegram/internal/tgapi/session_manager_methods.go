package tgapi

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/gotd/td/session"
	"github.com/rs/zerolog"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
)

type sessionManager struct {
	appID   int
	appHash string
	appCtx  context.Context

	sessions map[string]*LiveSession
	mu       sync.RWMutex
	logger   zerolog.Logger
}

func NewSessionManager(ctx context.Context, appID int, appHash string, baseLogger zerolog.Logger) SessionManager {
	logger := baseLogger.With().Str("component", "session_manager").Logger()

	return &sessionManager{
		appID:    appID,
		appHash:  appHash,
		appCtx:   ctx,
		sessions: make(map[string]*LiveSession),
		logger:   logger,
	}

}

func (sm *sessionManager) CreateSession(authCtx context.Context) SessionInfoCh {
	sessionID := uuid.New().String()

	hub := newBroadcastHub()
	hub.Start()

	ls := &LiveSession{

		sessionID: sessionID,
		appID:     sm.appID,
		appHash:   sm.appHash,
		logger:    sm.logger.With().Str("session_id", sessionID).Logger(),

		storage: new(session.StorageMemory),

		hub: hub,

		cmdCh:   make(chan commands, 64),
		qrCh:    make(chan string, 10),
		stateCh: make(chan SessionState, 10),
		errCh:   make(chan error, 1),
		passCh:  make(chan passReq, 4),
		closeCh: make(chan struct{}),
	}

	sm.mu.Lock()
	sm.sessions[sessionID] = ls
	sm.mu.Unlock()

	sCtx, cancel := context.WithCancel(sm.appCtx)
	ls.cancel = cancel

	go ls.run(sCtx, authCtx)

	return SessionInfoCh{
		SessionID: sessionID,
		QrChan:    ls.qrCh,
		ErrChan:   ls.errCh,
		StateChan: ls.stateCh,
	}

}

func (sm *sessionManager) SubmitPassword(sessionID, password string) error {
	sm.mu.RLock()
	session, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !ok {
		return errs.ErrSessionNotFound
	}

	resp := make(chan error, 1)

	select {

	case session.passCh <- passReq{
		password: password,
		resp:     resp,
	}:

	case <-sm.appCtx.Done():
		return sm.appCtx.Err()

	case <-session.closeCh:
		return errs.ErrSessionClosed
	}

	select {
	case err := <-resp:
		return err
	case <-sm.appCtx.Done():
		return sm.appCtx.Err()
	case <-session.closeCh:
		return errs.ErrSessionClosed
	}

}

func (sm *sessionManager) Get(sessionID string) Session {
	sm.mu.RLock()
	session, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()
	if !ok {
		return nil
	}
	return session
}

func (sm *sessionManager) Delete(sessionID string) error {
	sm.mu.Lock()
	session, ok := sm.sessions[sessionID]
	if !ok {
		sm.mu.Unlock()
		return errs.ErrSessionNotFound
	}
	delete(sm.sessions, sessionID)
	sm.mu.Unlock()

	session.close()

	fmt.Println("session to delete ->", sessionID)
	fmt.Println("all session ->", sm.sessions)
	return nil
}
