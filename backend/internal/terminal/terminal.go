package terminal

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev / panel access
	},
}

// WindowResizeMessage mewakili event resize xterm.js
type WindowResizeMessage struct {
	Type string `json:"type"` // "resize"
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// Session mewakili sesi terminal aktif
type Session struct {
	wsConn *websocket.Conn
	ptm    io.ReadWriteCloser
	mu     sync.Mutex
}

// HandleTerminalWebSocket menangani koneksi xterm.js melalui WebSocket
func HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Gagal upgrade websocket terminal: %v", err)
		return
	}
	defer ws.Close()

	// Ambil parameter target jail jika ada (misal: ?jail=web1 atau shell host utama)
	jailName := r.URL.Query().Get("jail")

	ptm, err := startPTY(jailName)
	if err != nil {
		_ = ws.WriteMessage(websocket.TextMessage, []byte("Gagal memulai sesi PTY: "+err.Error()+"\r\n"))
		return
	}
	defer ptm.Close()

	session := &Session{
		wsConn: ws,
		ptm:    ptm,
	}

	// Goroutine 1: Membaca output dari PTY lalu kirim ke WebSocket (Browser)
	errChan := make(chan error, 2)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := session.ptm.Read(buf)
			if err != nil {
				errChan <- err
				return
			}
			session.mu.Lock()
			err = session.wsConn.WriteMessage(websocket.BinaryMessage, buf[:n])
			session.mu.Unlock()
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Goroutine 2: Membaca input dari browser (WebSocket) lalu teruskan ke PTY
	go func() {
		for {
			msgType, data, err := session.wsConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}

			if msgType == websocket.TextMessage {
				// Cek apakah pesan adalah instruksi resize
				var resizeMsg WindowResizeMessage
				if err := json.Unmarshal(data, &resizeMsg); err == nil && resizeMsg.Type == "resize" {
					setPTYSize(session.ptm, resizeMsg.Cols, resizeMsg.Rows)
					continue
				}
			}

			// Tulis input keyboard ke PTY
			if _, err := session.ptm.Write(data); err != nil {
				errChan <- err
				return
			}
		}
	}()

	<-errChan
}
