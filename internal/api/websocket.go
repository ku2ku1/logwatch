package api

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" || origin == "http://localhost:5173" ||
			origin == "http://127.0.0.1:5173" ||
			origin == "http://localhost:5174" ||
			origin == "http://127.0.0.1:5174"
	},
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Hub struct {
	mu        sync.Mutex
	clients   map[*websocket.Conn]bool
	broadcast chan WSMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan WSMessage, 256),
	}
}

func (h *Hub) Run() {
	for msg := range h.broadcast {
		h.mu.Lock()
		for client := range h.clients {
			client.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if err := client.WriteJSON(msg); err != nil {
				log.Printf("[ws] client error, removing: %v", err)
				client.Close()
				delete(h.clients, client)
			}
		}
		h.mu.Unlock()
	}
}

func (h *Hub) addClient(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	count := len(h.clients)
	h.mu.Unlock()
	log.Printf("[ws] client connected (%d total)", count)
}

func (h *Hub) removeClient(conn *websocket.Conn) {
	h.mu.Lock()
	if _, ok := h.clients[conn]; ok {
		conn.Close()
		delete(h.clients, conn)
	}
	count := len(h.clients)
	h.mu.Unlock()
	log.Printf("[ws] client disconnected (%d total)", count)
}

func wsExtractToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	if cookie, err := r.Cookie("token"); err == nil {
		return cookie.Value
	}
	return ""
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := wsExtractToken(r)
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := s.jwt.Verify(token); err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	s.hub.addClient(conn)
	defer s.hub.removeClient(conn)

	since := time.Now().Add(-24 * time.Hour)
	if stats, err := s.db.GetStats(since); err == nil {
		conn.WriteJSON(WSMessage{Type: "stats", Data: stats})
	}
	if paths, err := s.db.GetTopPaths(since, 10); err == nil {
		conn.WriteJSON(WSMessage{Type: "top_paths", Data: paths})
	}
	if ips, err := s.db.GetTopIPs(since, 10); err == nil {
		conn.WriteJSON(WSMessage{Type: "top_ips", Data: ips})
	}

	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *Server) BroadcastUpdate() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		since := time.Now().Add(-24 * time.Hour)
		if stats, err := s.db.GetStats(since); err == nil {
			s.hub.broadcast <- WSMessage{Type: "stats", Data: stats}
		}
		if paths, err := s.db.GetTopPaths(since, 10); err == nil {
			s.hub.broadcast <- WSMessage{Type: "top_paths", Data: paths}
		}
		if ips, err := s.db.GetTopIPs(since, 10); err == nil {
			s.hub.broadcast <- WSMessage{Type: "top_ips", Data: ips}
		}
		if codes, err := s.db.GetStatusCodes(since); err == nil {
			s.hub.broadcast <- WSMessage{Type: "status_codes", Data: codes}
		}
	}
}
