package websocket

import (
	"sync"

	cws "github.com/coder/websocket"
)

// Hub manages websocket connections and group broadcasts.
type Hub struct {
	mu     sync.RWMutex
	conns  map[string]*Conn
	groups map[string]map[string]*Conn
}

func NewHub() *Hub {
	return &Hub{
		conns:  make(map[string]*Conn),
		groups: make(map[string]map[string]*Conn),
	}
}

func (h *Hub) Register(conn *Conn) {
	if h == nil || conn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn.ID()] = conn
}

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

func (h *Hub) Broadcast(msgType cws.MessageType, data []byte) {
	if h == nil {
		return
	}
	targets := h.snapshotConns()
	for _, conn := range targets {
		_ = conn.Send(msgType, data)
	}
}

func (h *Hub) BroadcastGroup(group string, msgType cws.MessageType, data []byte) {
	if h == nil || group == "" {
		return
	}
	targets := h.snapshotGroup(group)
	for _, conn := range targets {
		_ = conn.Send(msgType, data)
	}
}

// snapshotConns returns a copy of all registered connections under a short lock hold.
func (h *Hub) snapshotConns() []*Conn {
	h.mu.RLock()
	conns := make([]*Conn, 0, len(h.conns))
	for _, conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()
	return conns
}

// snapshotGroup returns a copy of connections in a group under a short lock hold.
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
