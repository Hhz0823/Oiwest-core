package adapters

import (
	"context"
	"io"
	"net"

	"github.com/Hhz0823/oiwest-core/app/common/protocol"
)

// Link represents a pair of reader/writer for proxy data transfer.
type Link struct {
	Reader io.Reader
	Writer io.Writer
}

// Inbound is the interface for all inbound handlers.
type Inbound interface {
	Tag() string
	Network() []string
	Start() error
	Close() error
}

// InboundHandler processes incoming connections.
type InboundHandler interface {
	Inbound
	Process(ctx context.Context, conn net.Conn, dispatcher InboundDispatcher) error
}

// InboundDispatcher dispatches inbound connections to outbound handlers.
type InboundDispatcher func(ctx context.Context, destination protocol.Destination, inbound Inbound) (OutboundHandler, error)

// OutboundHandler processes outgoing connections.
type OutboundHandler interface {
	Tag() string
	Process(ctx context.Context, link *Link, dialer NetDialer) error
	Close() error
}

// NetDialer is the interface for network dialing.
type NetDialer interface {
	Dial(ctx context.Context, destination protocol.Destination) (net.Conn, error)
}

// InboundManager manages all inbound handlers.
type InboundManager interface {
	Add(handler InboundHandler) error
	Remove(tag string) error
	Get(tag string) (InboundHandler, bool)
	Start() error
	Close() error
}

// OutboundManager manages all outbound handlers.
type OutboundManager interface {
	Add(handler OutboundHandler) error
	Remove(tag string) error
	Get(tag string) (OutboundHandler, bool)
	Default() OutboundHandler
	SetDefault(handler OutboundHandler)
}

// LinkGenerator creates new Link instances.
type LinkGenerator interface {
	Generate(ctx context.Context) *Link
}

// DefaultLinkGenerator creates Links using pipes.
type DefaultLinkGenerator struct{}

func (g *DefaultLinkGenerator) Generate(ctx context.Context) *Link {
	pr, pw := io.Pipe()
	return &Link{Reader: pr, Writer: pw}
}

