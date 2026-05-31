package dccp

import (
	"encoding/binary"
	"time"
)

type FeatureNegotiation struct {
	FeatureType   byte
	FeatureLength byte
	Value         []byte
	IsServer      bool
	IsConfirmed   bool
}

func BuildFeatureOption(feature byte, value []byte) DCCPOption {
	return DCCPOption{
		Type: OptFeature,
		Len:  byte(3 + len(value)),
		Data: append([]byte{feature, byte(len(value))}, value...),
	}
}

func BuildFeatureAckOption(feature byte) DCCPOption {
	return DCCPOption{
		Type: OptFeatureAck,
		Len:  3,
		Data: []byte{feature, 0},
	}
}

func BuildTimestampOption() DCCPOption {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(time.Now().UnixNano()/1000))
	return DCCPOption{
		Type: OptTimestamp,
		Len:  10,
		Data: b,
	}
}

func BuildTimestampEchoOption(ts []byte) DCCPOption {
	echo := make([]byte, 8)
	if len(ts) >= 8 {
		copy(echo, ts[:8])
	}
	return DCCPOption{
		Type: OptTimestampEcho,
		Len:  10,
		Data: echo,
	}
}

func BuildAckVectorOption(ackVector []byte) DCCPOption {
	return DCCPOption{
		Type: OptAckVector,
		Len:  byte(2 + len(ackVector)),
		Data: ackVector,
	}
}

func BuildDataChecksumOption() DCCPOption {
	return DCCPOption{
		Type: OptDataChecksum,
		Len:  2,
		Data: nil,
	}
}

func BuildChangeLOption(feature byte, value []byte) DCCPOption {
	return DCCPOption{
		Type: OptChangeL,
		Len:  byte(3 + len(value)),
		Data: append([]byte{feature, byte(len(value))}, value...),
	}
}

func BuildConfirmLOption(feature byte, value []byte) DCCPOption {
	return DCCPOption{
		Type: OptConfirmL,
		Len:  byte(3 + len(value)),
		Data: append([]byte{feature, byte(len(value))}, value...),
	}
}

func BuildChangeROption(feature byte, value []byte) DCCPOption {
	return DCCPOption{
		Type: OptChangeR,
		Len:  byte(3 + len(value)),
		Data: append([]byte{feature, byte(len(value))}, value...),
	}
}

func BuildConfirmROption(feature byte, value []byte) DCCPOption {
	return DCCPOption{
		Type: OptConfirmR,
		Len:  byte(3 + len(value)),
		Data: append([]byte{feature, byte(len(value))}, value...),
	}
}

func BuildElapsedTimeOption() DCCPOption {
	b := make([]byte, 4)
	return DCCPOption{
		Type: OptElapsedTime,
		Len:  6,
		Data: b,
	}
}

func BuildSlowReceiverOption(recvRate uint32) DCCPOption {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, recvRate)
	return DCCPOption{
		Type: OptSlowReceiver,
		Len:  6,
		Data: b,
	}
}

func BuildDataDroppedOption(dropCode byte, blockLength uint16) DCCPOption {
	b := make([]byte, 7)
	b[0] = dropCode
	binary.BigEndian.PutUint16(b[1:3], blockLength)
	return DCCPOption{
		Type: OptDataDropped,
		Len:  9,
		Data: b,
	}
}

func BuildMinCSUMOption() DCCPOption {
	return DCCPOption{
		Type: OptMinCSUM,
		Len:  2,
		Data: nil,
	}
}

func BuildInitCookieOption(cookie []byte, serviceCode ServiceCode) DCCPOption {
	data := make([]byte, 2+len(cookie)+4)
	data[0] = 0
	data[1] = 0
	copy(data[2:2+len(cookie)], cookie)
	copy(data[2+len(cookie):], serviceCode[:])
	return DCCPOption{
		Type: OptInitCookie,
		Len:  byte(2 + len(data)),
		Data: data,
	}
}
