package transport

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsReadBufSize  = 32 * 1024  // 32KB: matches proxy buffer size for efficient I/O
	wsWriteBufSize = 32 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   wsReadBufSize,
	WriteBufferSize:  wsWriteBufSize,
	CheckOrigin:      func(r *http.Request) bool { return true },
}

type WebSocketConn struct {
	conn   *websocket.Conn
	reader io.Reader
	mu     sync.Mutex
}

func NewWebSocketConn(tcpConn net.Conn, host, path string, settings *StreamSettings) (*WebSocketConn, error) {
	wsScheme := "ws"
	if settings != nil && settings.Security == "tls" {
		wsScheme = "wss"
	}

	headers := make(http.Header)
	if settings != nil && settings.WSSettings != nil {
		for k, v := range settings.WSSettings.Headers {
			headers.Set(k, v)
		}
		if settings.WSSettings.Host != "" {
			host = settings.WSSettings.Host
		}
	}
	u := fmt.Sprintf("%s://%s%s", wsScheme, host, path)

	dialer := &websocket.Dialer{
		NetDial: func(_, addr string) (net.Conn, error) {
			return tcpConn, nil
		},
		ReadBufferSize:   wsReadBufSize,
		WriteBufferSize:  wsWriteBufSize,
		HandshakeTimeout: 15 * time.Second,
	}

	wsConn, _, err := dialer.Dial(u, headers)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	return &WebSocketConn{
		conn: wsConn,
	}, nil
}

func (c *WebSocketConn) Read(p []byte) (int, error) {
	// Try to continue reading from previous message first
	if c.reader != nil {
		n, err := c.reader.Read(p)
		if err == io.EOF {
			c.reader = nil
			if n > 0 {
				return n, nil
			}
			// Fall through to read next message
		} else {
			return n, err
		}
	}

	_, r, err := c.conn.NextReader()
	if err != nil {
		return 0, err
	}

	n, err := r.Read(p)
	if err == io.EOF {
		// Message fully consumed in one read
		return n, nil
	}
	if err != nil {
		return n, err
	}

	// Partial read — save reader for subsequent calls
	c.reader = r
	return n, nil
}

func (c *WebSocketConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *WebSocketConn) Close() error {
	return c.conn.Close()
}

func (c *WebSocketConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *WebSocketConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *WebSocketConn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.conn.SetWriteDeadline(t)
}

func (c *WebSocketConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *WebSocketConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type wsResponseWriter struct {
	conn   net.Conn
	header http.Header
	mu     sync.Mutex
}

func (w *wsResponseWriter) Header() http.Header {
	return w.header
}

func (w *wsResponseWriter) Write(b []byte) (int, error) {
	return w.conn.Write(b)
}

func (w *wsResponseWriter) WriteHeader(statusCode int) {
}

type WebSocketListener struct {
	listener net.Listener
	settings *StreamSettings
	mu       sync.Mutex
}

func NewWebSocketListener(listener net.Listener, settings *StreamSettings) *WebSocketListener {
	return &WebSocketListener{
		listener: listener,
		settings: settings,
	}
}

func (l *WebSocketListener) Accept() (net.Conn, error) {
	tcpConn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}

	br := bufio.NewReader(tcpConn)
	_, err = br.Peek(4)
	if err != nil {
		tcpConn.Close()
		return nil, err
	}

	rw := &wsResponseWriter{
		conn:   tcpConn,
		header: make(http.Header),
	}

	req, err := http.ReadRequest(br)
	if err != nil {
		tcpConn.Close()
		return nil, err
	}

	wsConn, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		tcpConn.Close()
		return nil, err
	}

	return &WebSocketConn{
		conn: wsConn,
	}, nil
}

func (l *WebSocketListener) Close() error {
	return l.listener.Close()
}

func (l *WebSocketListener) Addr() net.Addr {
	return l.listener.Addr()
}
