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

var upgrader = websocket.Upgrader{
	ReadBufferSize:   4 * 1024,
	WriteBufferSize:  4 * 1024,
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
		ReadBufferSize:   4 * 1024,
		WriteBufferSize:  4 * 1024,
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
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	n := copy(p, message)
	return n, nil
}

func (c *WebSocketConn) Write(p []byte) (int, error) {
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
	return c.conn.SetReadDeadline(t)
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
