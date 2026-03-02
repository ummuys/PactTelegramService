package tgapi

import (
	"sync"
	"time"
)

type broadcastManager struct {
	mu       sync.RWMutex
	sessions map[string]*broadcastHub // айди сессии возвращает комнату слушателей
}

type broadcastHub struct {
	mu     sync.RWMutex
	subs   map[string]chan BroadcastMessage // айди слушателя возвращает канал слушателя
	closed bool
}

type BroadcastMessage struct {
	MessageID int64
	Text      string
	From      string
	Timestamp time.Time
}

func newBroadcastManager() *broadcastManager {
	return &broadcastManager{
		sessions: make(map[string]*broadcastHub),
	}
}

func (bm *broadcastManager) CreateHub(sessionID string) *broadcastHub {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if hub, ok := bm.sessions[sessionID]; ok {
		return hub
	}

	hub := &broadcastHub{
		subs: make(map[string]chan BroadcastMessage),
	}
	bm.sessions[sessionID] = hub
	return hub
}

func (bm *broadcastManager) GetHub(sessionID string) (*broadcastHub, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	hub, ok := bm.sessions[sessionID]
	return hub, ok
}

func (bm *broadcastManager) CloseSession(sessionID string) {
	bm.mu.Lock()
	hub, ok := bm.sessions[sessionID]
	if ok {
		delete(bm.sessions, sessionID)
	}
	bm.mu.Unlock()

	if ok {
		hub.Close()
	}
}

// -------- hub methods --------
func (h *broadcastHub) Subscribe(listernerID string) (<-chan BroadcastMessage, func(), bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, nil, false
	}

	c := make(chan BroadcastMessage, 100)
	h.subs[listernerID] = c

	unsub := func() {
		h.mu.Lock()
		ch, ok := h.subs[listernerID]
		if ok {
			delete(h.subs, listernerID)
			close(ch)
		}
		h.mu.Unlock()
	}

	return c, unsub, true
}

// Broadcast рассылает всем текущим подписчикам.
// Политика: не блокируемся, если подписчик медленный (дропаем сообщение для него).
func (h *broadcastHub) Broadcast(msg BroadcastMessage) bool {
	// snapshot каналов
	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		return false
	}

	chans := make([]chan BroadcastMessage, 0, len(h.subs))
	for _, ch := range h.subs {
		chans = append(chans, ch)
	}
	h.mu.RUnlock()

	// рассылка без лока
	for _, ch := range chans {
		select {
		case ch <- msg:
		default:

		}
	}

	return true
}

func (h *broadcastHub) Close() {

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}

	h.closed = true
	for id, ch := range h.subs {
		delete(h.subs, id)
		close(ch)
	}
	h.mu.Unlock()
}
