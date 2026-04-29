package protocol

import (
	"encoding/binary"
	"io"

	"github.com/sb-panel/dccp-kernel/common/buf"
)

const (
	HTTP  byte = 0
	TLS   byte = 1
	QUIC  byte = 2
	DCCP  byte = 3
	DNS   byte = 4
	SSH   byte = 5
	FTP   byte = 6
	SMTP  byte = 7
	Bittorrent byte = 8
	RDP   byte = 9
	Unknown byte = 99
)

type SniffResult struct {
	Protocol byte
	Domain   string
}

type Sniffer func(*buf.Buffer) (*SniffResult, error)

var sniffers []Sniffer

func RegisterSniffer(s Sniffer) {
	sniffers = append(sniffers, s)
}

func Sniff(b *buf.Buffer) (*SniffResult, error) {
	for _, s := range sniffers {
		r, err := s(b)
		if err != nil {
			continue
		}
		if r != nil {
			return r, nil
		}
	}
	return &SniffResult{Protocol: Unknown}, nil
}

type RequestHeader struct {
	Version byte
	Command byte
	Port    uint16
}

func ParseRequestHeader(r io.Reader) (*RequestHeader, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	return &RequestHeader{
		Version: hdr[0],
		Command: hdr[1],
		Port:    binary.BigEndian.Uint16(hdr[2:4]),
	}, nil
}

func WriteRequestHeader(w io.Writer, cmd byte, port uint16) error {
	hdr := [4]byte{1, cmd, 0, 0}
	binary.BigEndian.PutUint16(hdr[2:4], port)
	_, err := w.Write(hdr[:])
	return err
}

type AddressType byte

const (
	ATypeIPv4   AddressType = 1
	ATypeDomain AddressType = 2
	ATypeIPv6   AddressType = 3
)

func init() {
	RegisterSniffer(httpSniffer)
	RegisterSniffer(tlsSniffer)
	RegisterSniffer(dnsSniffer)
}

func httpSniffer(b *buf.Buffer) (*SniffResult, error) {
	if b.Len() < 4 {
		return nil, io.ErrUnexpectedEOF
	}
	data := b.Bytes()
	if len(data) >= 4 {
		if string(data[:4]) == "GET " || string(data[:4]) == "POST" ||
			string(data[:4]) == "HEAD" || string(data[:4]) == "PUT " ||
			string(data[:4]) == "DELE" || string(data[:4]) == "OPTI" ||
			string(data[:4]) == "CONN" || string(data[:4]) == "PATC" {
			return &SniffResult{Protocol: HTTP}, nil
		}
	}
	return nil, nil
}

func tlsSniffer(b *buf.Buffer) (*SniffResult, error) {
	if b.Len() < 6 {
		return nil, io.ErrUnexpectedEOF
	}
	data := b.Bytes()
	if data[0] == 0x16 && data[1] == 0x03 && data[2] >= 0x00 && data[2] <= 0x03 {
		if b.Len() >= 6+int(binary.BigEndian.Uint16(data[3:5])) {
			sni := extractSNI(data[5:])
			return &SniffResult{Protocol: TLS, Domain: sni}, nil
		}
	}
	return nil, nil
}

func dnsSniffer(b *buf.Buffer) (*SniffResult, error) {
	if b.Len() < 12 {
		return nil, io.ErrUnexpectedEOF
	}
	data := b.Bytes()
	if data[2]&0x80 == 0 && data[3]&0x0F == 0 {
		return &SniffResult{Protocol: DNS}, nil
	}
	return nil, nil
}

func extractSNI(data []byte) string {
	if len(data) < 5 {
		return ""
	}
	handshakeType := data[0]
	if handshakeType != 0x01 {
		return ""
	}
	length := int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	if len(data) < 4+length {
		return ""
	}
	payload := data[4 : 4+length]
	if len(payload) < 42 {
		return ""
	}
	offset := 2 + 32
	sessionIDLen := int(payload[offset])
	offset += 1 + sessionIDLen
	if offset+2 > len(payload) {
		return ""
	}
	cipherLen := int(binary.BigEndian.Uint16(payload[offset:]))
	offset += 2 + cipherLen
	if offset+1 > len(payload) {
		return ""
	}
	compLen := int(payload[offset])
	offset += 1 + compLen
	if offset+2 > len(payload) {
		return ""
	}
	extLen := int(binary.BigEndian.Uint16(payload[offset:]))
	offset += 2
	end := offset + extLen
	for offset+4 <= end && offset+4 <= len(payload) {
		extType := binary.BigEndian.Uint16(payload[offset:])
		extDataLen := int(binary.BigEndian.Uint16(payload[offset+2:]))
		offset += 4
		if extType == 0x0000 {
			if offset+extDataLen > end || offset+extDataLen > len(payload) {
				return ""
			}
			sniData := payload[offset : offset+extDataLen]
			if len(sniData) > 5 && sniData[0] == 0x00 {
				sniLen := int(binary.BigEndian.Uint16(sniData[3:]))
				if 5+sniLen <= len(sniData) {
					return string(sniData[5 : 5+sniLen])
				}
			}
			break
		}
		offset += extDataLen
	}
	return ""
}
