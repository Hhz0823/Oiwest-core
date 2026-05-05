package services

import (
	"encoding/json"
	"fmt"
	"strings"
)

func GenerateConfigJSON(node *ServerNode) string {
	config := map[string]interface{}{
		"log": map[string]interface{}{"loglevel": "warning"},
	}

	socksIn := map[string]interface{}{
		"tag": "socks-in", "port": 10808, "listen": "127.0.0.1",
		"protocol": "socks",
		"settings": map[string]interface{}{"auth": "noauth", "udp": true},
		"sniffing": map[string]interface{}{"enabled": true, "destOverride": []string{"http", "tls"}},
	}
	httpIn := map[string]interface{}{
		"tag": "http-in", "port": 10809, "listen": "127.0.0.1",
		"protocol": "http",
	}
	inbounds := []interface{}{socksIn, httpIn}

	if node.IPv6 {
		socks6 := map[string]interface{}{
			"tag": "socks-in6", "port": 10810, "listen": "::1",
			"protocol": "socks",
			"settings": map[string]interface{}{"auth": "noauth", "udp": true},
		}
		inbounds = append(inbounds, socks6)
	}

	config["inbounds"] = inbounds

	outbound := buildFullOutbound(node)
	directOut := map[string]interface{}{"tag": "direct", "protocol": "freedom"}
	outbounds := []interface{}{outbound, directOut}

	if node.IPv6 && node.MultiLine && node.Address6 != "" {
		out6 := buildFullOutbound(node)
		out6["tag"] = "proxy6"
		if sa, ok := out6["streamSettings"].(map[string]interface{}); ok {
			if tls, ok2 := sa["tlsSettings"].(map[string]interface{}); ok2 {
				tls["serverName"] = node.SNI
			}
		}
		if s, ok := out6["settings"].(map[string]interface{}); ok {
			updateOutboundAddress(s, node.Address6, node.Port6)
		}
		outbounds = append(outbounds, out6)
	}

	config["outbounds"] = outbounds

	routingRules := []interface{}{
		map[string]interface{}{"type": "field", "ip": []string{"geoip:private"}, "outboundTag": "direct"},
	}

	if node.IPv6 && node.MultiLine {
		routingRules = append(routingRules,
			map[string]interface{}{"type": "field", "ip": []string{"geoip:cn"}, "outboundTag": "direct"},
			map[string]interface{}{"type": "field", "network": "tcp,udp", "outboundTag": "proxy6"},
		)
	} else {
		routingRules = append(routingRules,
			map[string]interface{}{"type": "field", "ip": []string{"geoip:cn"}, "outboundTag": "direct"},
		)
	}

	config["routing"] = map[string]interface{}{
		"domainStrategy": "IPOnDemand",
		"rules":          routingRules,
	}

	dnsServers := []interface{}{map[string]interface{}{"address": "localhost"}}
	if node.IPv6 {
		dnsServers = append(dnsServers, map[string]interface{}{"address": "8.8.8.8"})
	}
	config["dns"] = map[string]interface{}{"servers": dnsServers}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

func buildFullOutbound(node *ServerNode) map[string]interface{} {
	out := map[string]interface{}{
		"tag":            "proxy",
		"protocol":       string(node.Protocol),
		"streamSettings": buildFullStreamSettings(node),
		"settings":       buildFullProtocolSettings(node),
	}
	if node.BBRType != "" && node.BBRType != "default" {
		out["bbr"] = map[string]interface{}{
			"enabled": true,
			"type":    node.BBRType,
		}
	}
	return out
}

func buildFullStreamSettings(node *ServerNode) map[string]interface{} {
	s := map[string]interface{}{}

	if node.Network == "" {
		node.Network = "tcp"
	}
	s["network"] = node.Network

	if node.Security != "" {
		s["security"] = node.Security
	}

	if node.TLS || node.Security == "tls" || node.Security == "reality" {
		s["security"] = node.Security
		if s["security"] == "" || s["security"] == "none" {
			s["security"] = "tls"
		}

		tlsS := map[string]interface{}{
			"serverName":    node.SNI,
			"allowInsecure": node.AllowInsecure,
		}
		if node.Fingerprint != "" {
			tlsS["fingerprint"] = node.Fingerprint
		}
		if node.TLSCertFile != "" {
			tlsS["certificateFile"] = node.TLSCertFile
		}
		if node.TLSKeyFile != "" {
			tlsS["keyFile"] = node.TLSKeyFile
		}

		s["tlsSettings"] = tlsS
	}

	if node.Security == "reality" {
		delete(s, "tlsSettings")
		s["realitySettings"] = map[string]interface{}{
			"serverName":  node.SNI,
			"fingerprint": node.Fingerprint,
			"publicKey":   node.PublicKey,
			"shortId":     node.ShortID,
			"spiderX":     node.SpiderX,
		}
	}

	switch node.Network {
	case "tcp":
		if node.HeaderType != "" && node.HeaderType != "none" {
			tcpS := map[string]interface{}{"header": map[string]interface{}{"type": node.HeaderType}}
			if node.HeaderType == "http" && node.Host != "" {
				tcpS["header"] = map[string]interface{}{"type": "http", "request": map[string]interface{}{
					"headers": map[string]interface{}{"Host": strings.Split(node.Host, ",")},
				}}
			}
			s["tcpSettings"] = tcpS
		}
		if node.DsUseDomain {
			if _, ok := s["tcpSettings"]; !ok {
				s["tcpSettings"] = map[string]interface{}{}
			}
			s["tcpSettings"].(map[string]interface{})["header"] = map[string]interface{}{"type": "http"}
		}
	case "ws":
		s["wsSettings"] = map[string]interface{}{
			"path":    node.Path,
			"headers": map[string]interface{}{"Host": node.Host},
		}
	case "grpc":
		s["grpcSettings"] = map[string]interface{}{"serviceName": node.Path}
	case "quic":
		qs := map[string]interface{}{
			"security": node.QuicSecurity,
			"key":      node.QuicKey,
			"header":   map[string]interface{}{"type": node.HeaderType},
		}
		if qs["security"] == "" {
			qs["security"] = "none"
		}
		if qs["header"].(map[string]interface{})["type"] == "" {
			qs["header"].(map[string]interface{})["type"] = "none"
		}
		s["quicSettings"] = qs
	case "dccp":
		s["dccpSettings"] = map[string]interface{}{
			"ccid":          4,
			"serviceCode":   "V2RY",
			"maxPacketSize": 1500,
		}
	case "mkcp":
		ms := map[string]interface{}{
			"mtu":              1350,
			"tti":              50,
			"uplinkCapacity":   5,
			"downlinkCapacity": 20,
			"congestion":       false,
			"readBufferSize":   2,
			"writeBufferSize":  2,
		}
		if node.MkcpHeader != "" {
			ms["header"] = map[string]interface{}{"type": node.MkcpHeader}
		}
		if node.MkcpSeed != "" {
			ms["seed"] = node.MkcpSeed
		}
		s["kcpSettings"] = ms
	case "h2":
		s["httpSettings"] = map[string]interface{}{
			"path": node.H2Path,
			"host": strings.Split(node.H2Host, ","),
		}
	case "xhttp":
		xs := map[string]interface{}{}
		if node.XhttpMode != "" {
			xs["mode"] = node.XhttpMode
		}
		if node.XhttpPath != "" {
			xs["path"] = node.XhttpPath
		}
		if node.Host != "" {
			xs["host"] = node.Host
		}
		s["xhttpSettings"] = xs
	}

	return s
}

func buildFullProtocolSettings(node *ServerNode) map[string]interface{} {
	switch node.Protocol {
	case ProtocolVMess:
		return map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{"address": node.Address, "port": node.Port, "users": []interface{}{
					map[string]interface{}{"id": node.UUID, "security": "auto", "alterId": 0},
				}},
			},
		}
	case ProtocolVLess:
		return map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{"address": node.Address, "port": node.Port, "users": []interface{}{
					map[string]interface{}{"id": node.UUID, "flow": node.Flow, "encryption": "none"},
				}},
			},
		}
	case ProtocolTrojan:
		return map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{"address": node.Address, "port": node.Port, "password": node.Password},
			},
		}
	case ProtocolShadowsocks:
		method := node.SSMethod
		if method == "" {
			method = "aes-256-gcm"
		}
		return map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{"address": node.Address, "port": node.Port, "password": node.Password, "method": method},
			},
		}
	case ProtocolSOCKS:
		return map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{"address": node.Address, "port": node.Port},
			},
		}
	case ProtocolHTTP:
		return map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{"address": node.Address, "port": node.Port},
			},
		}
	case ProtocolWireGuard:
		return map[string]interface{}{
			"secretKey": node.WireguardPriv,
			"peers":     []interface{}{map[string]interface{}{"endpoint": fmt.Sprintf("%s:%d", node.Address, node.Port), "publicKey": node.WireguardPub}},
		}
	case ProtocolDNS:
		return map[string]interface{}{
			"network": node.Network,
			"address": node.Address,
			"port":    node.Port,
		}
	case ProtocolFreedom:
		return map[string]interface{}{"domainStrategy": "AsIs"}
	case ProtocolBlackhole:
		return map[string]interface{}{"response": map[string]interface{}{"type": "none"}}
	default:
		return map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{"address": node.Address, "port": node.Port, "users": []interface{}{
					map[string]interface{}{"id": node.UUID},
				}},
			},
		}
	}
}

func updateOutboundAddress(s map[string]interface{}, addr string, port int) {
	vnext, _ := s["vnext"].([]interface{})
	if len(vnext) > 0 {
		if m, ok := vnext[0].(map[string]interface{}); ok {
			m["address"] = addr
			if port > 0 {
				m["port"] = port
			}
		}
	}
	servers, _ := s["servers"].([]interface{})
	if len(servers) > 0 {
		if m, ok := servers[0].(map[string]interface{}); ok {
			m["address"] = addr
			if port > 0 {
				m["port"] = port
			}
		}
	}
}

func ParseVmessLink(link string) (*ServerNode, error) {
	if !strings.HasPrefix(link, "vmess://") {
		return nil, fmt.Errorf("不支持的链接格式")
	}
	base64Str := strings.TrimPrefix(link, "vmess://")
	decoded, err := decodeBase64(base64Str)
	if err != nil {
		return nil, fmt.Errorf("Base64解码失败: %w", err)
	}
	var vmData map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &vmData); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}
	node := &ServerNode{Protocol: ProtocolVMess, Network: "tcp", Security: "auto"}
	if v, ok := vmData["ps"].(string); ok {
		node.Name = v
	}
	if v, ok := vmData["add"].(string); ok {
		node.Address = v
	}
	if v, ok := vmData["port"]; ok {
		switch p := v.(type) {
		case float64:
			node.Port = int(p)
		case string:
			fmt.Sscanf(p, "%d", &node.Port)
		}
	}
	if v, ok := vmData["id"].(string); ok {
		node.UUID = v
	}
	if v, ok := vmData["net"].(string); ok {
		node.Network = v
	}
	if v, ok := vmData["host"].(string); ok {
		node.Host = v
	}
	if v, ok := vmData["path"].(string); ok {
		node.Path = v
	}
	if v, ok := vmData["tls"].(string); ok {
		node.TLS = v == "tls"
	}
	if v, ok := vmData["sni"].(string); ok {
		node.SNI = v
	}
	return node, nil
}

func decodeBase64(s string) (string, error) {
	padding := (4 - len(s)%4) % 4
	s += strings.Repeat("=", padding)
	result := make([]byte, len(s))
	n, buf, bits := 0, uint32(0), 0
	for _, c := range s {
		switch {
		case c >= 'A' && c <= 'Z':
			buf = (buf << 6) | uint32(c-'A')
		case c >= 'a' && c <= 'z':
			buf = (buf << 6) | uint32(c-'a'+26)
		case c >= '0' && c <= '9':
			buf = (buf << 6) | uint32(c-'0'+52)
		case c == '+':
			buf = (buf << 6) | 62
		case c == '/':
			buf = (buf << 6) | 63
		case c == '=':
			break
		default:
			continue
		}
		bits += 6
		if bits >= 8 {
			bits -= 8
			result[n] = byte(buf >> bits)
			n++
			buf &^= 0xFF << bits
		}
	}
	return string(result[:n]), nil
}
