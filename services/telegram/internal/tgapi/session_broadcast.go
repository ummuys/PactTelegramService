package tgapi

import (
	"sync"
	"time"
)

type BroadcastMessage struct {
	MessageID int64
	Text      string
	From      string
	Timestamp time.Time
}

type broadcastHub struct {
	cmds chan hubCmd
	done chan struct{}

	startOnce sync.Once
	closeOnce sync.Once
}

type hubCmd interface {
	run(h *hubState)
}

type hubState struct {
	subs   map[string]chan BroadcastMessage
	closed bool
}

// ---------- constructor ----------

func newBroadcastHub() *broadcastHub {
	return &broadcastHub{
		cmds: make(chan hubCmd, 64),
		done: make(chan struct{}),
	}
}

func (h *broadcastHub) Start() {
	h.startOnce.Do(func() {
		go h.loop()
	})
}

func (h *broadcastHub) loop() {
	st := &hubState{subs: make(map[string]chan BroadcastMessage)}

	for cmd := range h.cmds {
		cmd.run(st)

		if st.closed {
			for _, ch := range st.subs {
				close(ch)
			}
			close(h.done)
			return
		}
	}

	for _, ch := range st.subs {
		close(ch)
	}

	close(h.done)
}

func (h *broadcastHub) Done() <-chan struct{} { return h.done }

func (h *broadcastHub) Close() {
	h.closeOnce.Do(func() {
		h.sendControl(&cmdClose{})
	})
}

// ---------- public API ----------

func (h *broadcastHub) Subscribe(listenerID string) <-chan BroadcastMessage {
	ch := make(chan BroadcastMessage, 16)
	h.sendControl(&cmdSubscribe{id: listenerID, ch: ch})
	return ch
}

func (h *broadcastHub) Unsubscribe(listenerID string) {
	h.sendControl(&cmdUnsubscribe{id: listenerID})
}

func (h *broadcastHub) Broadcast(msg BroadcastMessage) {
	h.sendBroadcast(&cmdBroadcast{msg: msg})
}

// ----------  helpers ----------

func (h *broadcastHub) sendControl(cmd hubCmd) {
	select {
	case <-h.done:
		return
	case h.cmds <- cmd:
	}
}

func (h *broadcastHub) sendBroadcast(cmd hubCmd) {
	select {
	case <-h.done:
		return
	case h.cmds <- cmd:
	default:
	}
}

// ---------- commands ----------

type cmdSubscribe struct {
	id string
	ch chan BroadcastMessage
}

func (c *cmdSubscribe) run(h *hubState) {
	if h.closed {
		close(c.ch)
		return
	}

	// повторная подписка на тот же id: закрываем старый канал, чтобы старый клиент не висел
	if old, ok := h.subs[c.id]; ok {
		close(old)
	}
	h.subs[c.id] = c.ch
}

type cmdUnsubscribe struct {
	id string
}

func (c *cmdUnsubscribe) run(h *hubState) {
	if ch, ok := h.subs[c.id]; ok {
		delete(h.subs, c.id)
		close(ch)
	}
}

type cmdBroadcast struct {
	msg BroadcastMessage
}

func (c *cmdBroadcast) run(h *hubState) {
	if h.closed {
		return
	}
	for _, ch := range h.subs {
		select {
		case ch <- c.msg:
		default:
		}
	}
}

type cmdClose struct{}

func (c *cmdClose) run(h *hubState) {
	h.closed = true
}
