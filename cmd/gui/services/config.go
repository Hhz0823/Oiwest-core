package services

import (
	"encoding/json"
	"fmt"
	"strings"
)

func GenerateConfigJSON(node *ServerNode) string {
	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
	}

	inbound := map[string]interface{}{
		"tag":      "socks-in",
		"port":     10808,
		"listen":   "127.0.0.1",
		"protocol": "socks",
		"settings": map[string]interface{}{
			"auth": "noauth",
			"udp":  true,
		},
		"sniffing": map[string]interface{}{
			"enabled":      true,
			"destOverride": []string{"http", "tls"},
		},
	}

	httpInbound := map[string]interface{}{
		"tag":      "http-in",
		"port":     10809,
		"listen":   "127.0.0.1",
		"protocol": "http",
	}

	config["inbounds"] = []interface{}{inbound, httpInbound}

	outbound := map[string]interface{}{
		"tag":      "proxy",
		"protocol": string(node.Protocol),
	}

	streamSettings := buildStreamSettings(node)
	protocolSettings := buildProtocolSettings(node)

	outbound["streamSettings"] = streamSettings
	outbound["settings"] = protocolSettings

	directOutbound := map[string]interface{}{
		"tag":      "direct",
		"protocol": "freedom",
	}

	config["outbounds"] = []interface{}{outbound, directOutbound}

	routing := map[string]interface{}{
		"domainStrategy": "IPOnDemand",
		"rules": []interface{}{
			map[string]interface{}{
				"type":        "field",
				"ip":          []string{"geoip:private"},
				"outboundTag": "direct",
			},
			map[string]interface{}{
				"type":        "field",
				"ip":          []string{"geoip:cn"},
				"outboundTag": "direct",
			},
		},
	}
	config["routing"] = routing

	config["dns"] = map[string]interface{}{
		"servers": []interface{}{
			map[string]interface{}{
				"address": "localhost",
			},
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

func buildStreamSettings(node *ServerNode) map[string]interface{} {
	settings := map[string]interface{}{
		"network":  node.Network,
		"security": node.Security,
	}

	if node.Network == "" {
		settings["network"] = "tcp"
	}

	if node.TLS {
		if node.Security == "" {
			settings["security"] = "tls"
		}
		tlsSettings := map[string]interface{}{
			"serverName":    node.SNI,
			"allowInsecure": node.AllowInsecure,
		}
		if node.Fingerprint != "" {
			tlsSettings["fingerprint"] = node.Fingerprint
		}
		settings["tlsSettings"] = tlsSettings
	}

	if node.Security == "reality" {
		realitySettings := map[string]interface{}{
			"serverName":  node.SNI,
			"fingerprint": node.Fingerprint,
			"publicKey":   node.PublicKey,
			"shortId":     node.ShortID,
			"spiderX":     node.SpiderX,
		}
		settings["realitySettings"] = realitySettings
	}

	switch node.Network {
	case "ws":
		settings["wsSettings"] = map[string]interface{}{
			"path": node.Path,
			"headers": map[string]interface{}{
				"Host": node.Host,
			},
		}
	case "grpc":
		settings["grpcSettings"] = map[string]interface{}{
			"serviceName": node.Path,
		}
	case "dccp":
		settings["dccpSettings"] = map[string]interface{}{
			"ccid":         4,
			"serviceCode":  "V2RY",
			"maxPacketSize": 1500,
		}
	}

	return settings
}

func buildProtocolSettings(node *ServerNode) map[string]interface{} {
	switch node.Protocol {
	case ProtocolVMess:
		return map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": node.Address,
					"port":    node.Port,
					"users": []interface{}{
						map[string]interface{}{
							"id":       node.UUID,
							"security": "auto",
						},
					},
				},
			},
		}
	case ProtocolVLess:
		return map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": node.Address,
					"port":    node.Port,
					"users": []interface{}{
						map[string]interface{}{
							"id":   node.UUID,
							"flow": node.Flow,
						},
					},
				},
			},
		}
	case ProtocolTrojan:
		return map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{
					"address":  node.Address,
					"port":     node.Port,
					"password": node.Password,
				},
			},
		}
	case ProtocolShadowsocks:
		return map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{
					"address":  node.Address,
					"port":     node.Port,
					"password": node.Password,
					"method":   "aes-256-gcm",
				},
			},
		}
	case ProtocolSOCKS:
		return map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{
					"address": node.Address,
					"port":    node.Port,
				},
			},
		}
	default:
		return map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": node.Address,
					"port":    node.Port,
					"users": []interface{}{
						map[string]interface{}{
							"id": node.UUID,
						},
					},
				},
			},
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

	node := &ServerNode{
		Protocol: ProtocolVMess,
		Network:  "tcp",
		Security: "auto",
	}

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
	n := 0
	buf := uint32(0)
	bits := 0

	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			buf = (buf << 6) | uint32(c-'A')
		} else if c >= 'a' && c <= 'z' {
			buf = (buf << 6) | uint32(c-'a'+26)
		} else if c >= '0' && c <= '9' {
			buf = (buf << 6) | uint32(c-'0'+52)
		} else if c == '+' {
			buf = (buf << 6) | 62
		} else if c == '/' {
			buf = (buf << 6) | 63
		} else if c == '=' {
			break
		} else {
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
