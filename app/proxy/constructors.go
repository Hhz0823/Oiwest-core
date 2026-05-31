package proxy

import (
	"encoding/json"

	netaddr "github.com/Hhz0823/oiwest-core/app/common/net"
	"github.com/Hhz0823/oiwest-core/app/transport"
)

// Inbound constructor functions

func NewDCCPInboundHandler(tag string, port uint16, listen string, ss *transport.StreamSettings, mgr *ProxyManager) *DCCPInboundHandler {
	return &DCCPInboundHandler{
		tag:      tag,
		port:     port,
		listen:   listen,
		settings: ss,
		manager:  mgr,
	}
}

func NewVmessInboundHandler(tag string, port uint16, listen string, settings json.RawMessage, ss *transport.StreamSettings, mgr *ProxyManager) *VmessInboundHandler {
	return &VmessInboundHandler{
		tag:            tag,
		port:           port,
		listen:         listen,
		settings:       settings,
		streamSettings: ss,
		manager:        mgr,
	}
}

func NewVlessInboundHandler(tag string, port uint16, listen string, settings json.RawMessage, ss *transport.StreamSettings, mgr *ProxyManager) *VlessInboundHandler {
	return &VlessInboundHandler{
		tag:            tag,
		port:           port,
		listen:         listen,
		settings:       settings,
		streamSettings: ss,
		manager:        mgr,
	}
}

func NewTrojanInboundHandler(tag string, port uint16, listen string, settings json.RawMessage, ss *transport.StreamSettings, mgr *ProxyManager) *TrojanInboundHandler {
	return &TrojanInboundHandler{
		tag:            tag,
		port:           port,
		listen:         listen,
		settings:       settings,
		streamSettings: ss,
		manager:        mgr,
	}
}

func NewShadowsocksInboundHandler(tag string, port uint16, listen string, settings json.RawMessage, ss *transport.StreamSettings, mgr *ProxyManager) *ShadowsocksInboundHandler {
	return &ShadowsocksInboundHandler{
		tag:            tag,
		port:           port,
		listen:         listen,
		settings:       settings,
		streamSettings: ss,
		manager:        mgr,
	}
}

// Outbound constructor functions

func NewDirectOutboundHandler(tag string) *DirectOutboundHandler {
	return &DirectOutboundHandler{tag: tag}
}

func NewFreedomOutboundHandler(tag string) *FreedomOutboundHandler {
	return &FreedomOutboundHandler{tag: tag}
}

func NewBlackholeOutboundHandler(tag string) *BlackholeOutboundHandler {
	return &BlackholeOutboundHandler{tag: tag}
}

func NewDNSOutboundHandler(tag, network, address string, port uint16) *DNSOutboundHandler {
	return &DNSOutboundHandler{
		tag:     tag,
		network: network,
		address: address,
		port:    port,
	}
}

func NewSocksOutboundHandler(tag string) *SocksOutboundHandler {
	return &SocksOutboundHandler{tag: tag}
}

func NewDCCPOutboundHandler(tag string, target netaddr.Address, ss *transport.StreamSettings) *DCCPOutboundHandler {
	return &DCCPOutboundHandler{
		tag:            tag,
		target:         target,
		streamSettings: ss,
	}
}

