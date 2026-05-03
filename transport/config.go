package transport

import (
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/Hhz0823/oiwest-core/transport/dccp"
)

var (
	ErrInvalidTransport = errors.New("invalid transport configuration")
	ErrTransportClosed  = errors.New("transport connection closed")
)

type TransportProtocol string

const (
	TransportTCP     TransportProtocol = "tcp"
	TransportDCCP    TransportProtocol = "dccp"
	TransportDCCPOverUDP TransportProtocol = "dccpou"
	TransportWebSocket TransportProtocol = "ws"
	TransportHTTP2    TransportProtocol = "h2"
	TransportGRPC     TransportProtocol = "grpc"
	TransportQUIC     TransportProtocol = "quic"
	TransportDCCPOverQUIC TransportProtocol = "dccpq"
)

type StreamSettings struct {
	Network          TransportProtocol `json:"network"`
	Security         string            `json:"security"`
	TLSSettings      *TLSSettings      `json:"tlsSettings"`
	TCPSettings      *TCPSettings      `json:"tcpSettings"`
	DCCPSettings     *DCCPSettings     `json:"dccpSettings"`
	WSSettings       *WSSettings       `json:"wsSettings"`
	HTTP2Settings    *HTTP2Settings    `json:"http2Settings"`
	GRPCSettings     *GRPCSettings     `json:"grpcSettings"`
	QUICSettings     *QUICSettings     `json:"quicSettings"`
	Sockopt          *Sockopt          `json:"sockopt"`
	DialerProxy      string            `json:"dialerProxy"`
}

type TLSSettings struct {
	ServerName        string   `json:"serverName"`
	AllowInsecure     bool     `json:"allowInsecure"`
	ALPN              []string `json:"alpn"`
	Certificates      []Certificate `json:"certificates"`
	VerifyClientCert  bool     `json:"verifyClientCert"`
	RejectUnknownSNI  bool     `json:"rejectUnknownSni"`
	PinnedPeerCertChainSHA256 [][]byte `json:"pinnedPeerCertificateChainSha256"`
}

type Certificate struct {
	Certificate      []string `json:"certificate"`
	Key              []string `json:"key"`
	Usage            string   `json:"usage"`
	Issuer           string   `json:"issuer"`
	CertificateFile  string   `json:"certificateFile"`
	KeyFile          string   `json:"keyFile"`
}

type TCPSettings struct {
	Header           *TCPHeader    `json:"header"`
	AcceptProxyProtocol bool       `json:"acceptProxyProtocol"`
}

type TCPHeader struct {
	Type    string `json:"type"`
	Request *HTTPRequestOpt `json:"request"`
	Response *HTTPResponseOpt `json:"response"`
}

type HTTPRequestOpt struct {
	Version string              `json:"version"`
	Method  string              `json:"method"`
	Path    []string            `json:"path"`
	Headers map[string][]string `json:"headers"`
}

type HTTPResponseOpt struct {
	Version string              `json:"version"`
	Status  string              `json:"status"`
	Reason  string              `json:"reason"`
	Headers map[string][]string `json:"headers"`
}

type DCCPSettings struct {
	CCID             dccp.CCID     `json:"ccid"`
	ServiceCode      string        `json:"serviceCode"`
	MaxPacketSize    int           `json:"maxPacketSize"`
	HandshakeTimeout time.Duration `json:"handshakeTimeout"`
	MaxRetries       int           `json:"maxRetries"`
	EnableObfuscation bool         `json:"enableObfuscation"`
	ObfuscationType  string        `json:"obfuscationType"`
	Padding          *PaddingSettings `json:"padding"`
}

type PaddingSettings struct {
	Enabled   bool   `json:"enabled"`
	Strategy  string `json:"strategy"`
	MinSize   int    `json:"minSize"`
	MaxSize   int    `json:"maxSize"`
}

type WSSettings struct {
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Host    string            `json:"host"`
}

type HTTP2Settings struct {
	Host string   `json:"host"`
	Path string   `json:"path"`
}

type GRPCSettings struct {
	ServiceName string `json:"serviceName"`
	MultiMode   bool   `json:"multiMode"`
}

type QUICSettings struct {
	Security string `json:"security"`
	Key      string `json:"key"`
	Header   *TCPHeader `json:"header"`
}

type Sockopt struct {
	Mark         int32           `json:"mark"`
	TOS          int32           `json:"tos"`
	TCPKeepAlive int32           `json:"tcpKeepAliveInterval"`
	TCPFastOpen  bool            `json:"tcpFastOpen"`
	TProxy       string          `json:"tproxy"`
	TCPUserTimeout int32         `json:"tcpUserTimeout"`
	DomainStrategy string        `json:"domainStrategy"`
	DialerProxy    string        `json:"dialerProxy"`
	BindAddress    string        `json:"bindAddress"`
	BindPort      uint16         `json:"bindPort"`
	Interface      string        `json:"interface"`
}

func DefaultStreamSettings() *StreamSettings {
	return &StreamSettings{
		Network: TransportDCCP,
		Security: "none",
		DCCPSettings: &DCCPSettings{
			CCID:            dccp.CCID4,
			ServiceCode:     "V2RY",
			MaxPacketSize:   1500,
			HandshakeTimeout: 15 * time.Second,
			MaxRetries:      3,
			EnableObfuscation: true,
			ObfuscationType:  "random",
		},
	}
}

func ParseStreamSettings(data []byte) (*StreamSettings, error) {
	settings := DefaultStreamSettings()
	if err := json.Unmarshal(data, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *StreamSettings) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

type TransportListener interface {
	Listen(addr net.Addr) (net.Listener, error)
	ListenPacket(addr net.Addr) (net.PacketConn, error)
	Close() error
	Addr() net.Addr
}

type TransportDialer interface {
	Dial(addr net.Addr) (net.Conn, error)
}

type TransportHandler interface {
	NewConnection(conn net.Conn) (net.Conn, error)
}
