package websocket

import (
	"sync"

	cws "github.com/coder/websocket"
)

// Hub manages websocket connections and group broadcasts within one process.
// It is not a cross-instance session store.
type Hub struct {
	mu     sync.RWMutex
	conns  map[string]*Conn
	groups map[string]map[string]*Conn
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{
		conns:  make(map[string]*Conn),
		groups: make(map[string]map[string]*Conn),
	}
}

// Register adds conn to the hub.
func (h *Hub) Register(conn *Conn) {
	if h == nil || conn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn.ID()] = conn
}

// Unregister removes conn from the hub and all groups.
func (h *Hub) Unregister(conn *Conn) {
	if h == nil || conn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, conn.ID())
	for _, members := range h.groups {
		delete(members, conn.ID())
	}
}

// Join adds conn to a named group.
func (h *Hub) Join(group string, conn *Conn) {
	if h == nil || conn == nil || group == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	members, ok := h.groups[group]
	if !ok {
		members = make(map[string]*Conn)
		h.groups[group] = members
	}
	members[conn.ID()] = conn
}

// Leave removes conn from a named group.
func (h *Hub) Leave(group string, conn *Conn) {
	if h == nil || conn == nil || group == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if members, ok := h.groups[group]; ok {
		delete(members, conn.ID())
		if len(members) == 0 {
			delete(h.groups, group)
		}
	}
}

// Broadcast non-blocking-enqueues to every registered connection.
// Returns how many targets failed (full queue, closed, etc.). A slow peer
// never blocks delivery to others.
func (h *Hub) Broadcast(msgType cws.MessageType, data []byte) (failed int) {
	if h == nil {
		return 0
	}
	return h.fanout(h.snapshotConns(), msgType, data)
}

// BroadcastGroup is Broadcast restricted to one group.
func (h *Hub) BroadcastGroup(group string, msgType cws.MessageType, data []byte) (failed int) {
	if h == nil || group == "" {
		return 0
	}
	return h.fanout(h.snapshotGroup(group), msgType, data)
}

func (h *Hub) fanout(targets []*Conn, msgType cws.MessageType, data []byte) int {
	failed := 0
	for _, conn := range targets {
		if err := conn.TrySend(msgType, data); err != nil {
			failed++
		}
	}
	return failed
}

// Len returns the number of registered connections.
func (h *Hub) Len() int {
	if h == nil {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

func (h *Hub) snapshotConns() []*Conn {
	h.mu.RLock()
	conns := make([]*Conn, 0, len(h.conns))
	for _, conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()
	return conns
}

func (h *Hub) snapshotGroup(group string) []*Conn {
	h.mu.RLock()
	members := h.groups[group]
	conns := make([]*Conn, 0, len(members))
	for _, conn := range members {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()
	return conns
}
