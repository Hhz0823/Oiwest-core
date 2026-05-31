package dccp

import (
	"encoding/binary"
	"errors"
	"io"
)

var (
	ErrPacketTooSmall    = errors.New("dccp: packet too small")
	ErrInvalidChecksum   = errors.New("dccp: invalid checksum")
	ErrInvalidPacketType = errors.New("dccp: invalid packet type")
	ErrInvalidOption     = errors.New("dccp: invalid option")
)

type DCCPOption struct {
	Type OptionType
	Len  byte
	Data []byte
}

func EncodeDCCPPacket(hdr *DCCPHeader, options []DCCPOption, payload []byte) ([]byte, error) {
	optionSize := 0
	for _, opt := range options {
		optionSize += int(opt.Len)
	}
	paddingSize := (4 - (optionSize % 4)) % 4
	totalOptionsSize := optionSize + paddingSize

	dataOffset := byte((HeaderSize + totalOptionsSize) / 4)
	if dataOffset > 15 {
		dataOffset = 15
	}

	totalSize := HeaderSize + totalOptionsSize + len(payload)
	buf := make([]byte, totalSize)

	binary.BigEndian.PutUint16(buf[0:2], hdr.SourcePort)
	binary.BigEndian.PutUint16(buf[2:4], hdr.DestPort)

	buf[4] = (dataOffset << 4) | (hdr.CCVal & 0x0F)
	buf[5] = (byte(hdr.Type) & 0x0F) | ((hdr.Extended & 0x01) << 7)

	binary.BigEndian.PutUint16(buf[6:8], hdr.Checksum)

	if hdr.Extended == 1 {
		binary.BigEndian.PutUint64(buf[8:16], hdr.SequenceNumber)
	}

	offset := HeaderSize
	for _, opt := range options {
		buf[offset] = byte(opt.Type)
		buf[offset+1] = opt.Len
		copy(buf[offset+2:offset+int(opt.Len)], opt.Data)
		offset += int(opt.Len)
	}

	for i := 0; i < paddingSize; i++ {
		buf[offset+i] = byte(OptPadding)
	}
	offset += paddingSize

	copy(buf[offset:], payload)

	checksum := computeChecksum(buf)
	binary.BigEndian.PutUint16(buf[6:8], checksum)

	return buf, nil
}

func DecodeDCCPPacket(data []byte) (*DCCPHeader, []DCCPOption, []byte, error) {
	if len(data) < HeaderSize {
		return nil, nil, nil, ErrPacketTooSmall
	}

	hdr := &DCCPHeader{
		SourcePort: binary.BigEndian.Uint16(data[0:2]),
		DestPort:   binary.BigEndian.Uint16(data[2:4]),
		DataOffset: (data[4] >> 4) & 0x0F,
		CCVal:      data[4] & 0x0F,
	}

	hdr.Type = PacketType(data[5] & 0x0F)
	hdr.Extended = (data[5] >> 7) & 0x01

	hdr.Checksum = binary.BigEndian.Uint16(data[6:8])
	hdr.ChecksumCoverage = 0

	if hdr.Extended == 1 {
		if len(data) < 16 {
			return nil, nil, nil, ErrPacketTooSmall
		}
		hdr.SequenceNumber = binary.BigEndian.Uint64(data[8:16])
	}

	storedChecksum := hdr.Checksum
	binary.BigEndian.PutUint16(data[6:8], 0)
	computedChecksum := computeChecksum(data)
	binary.BigEndian.PutUint16(data[6:8], storedChecksum)

	if storedChecksum != 0 && storedChecksum != computedChecksum {
	}

	headerEnd := int(hdr.DataOffset) * 4
	if headerEnd > len(data) {
		headerEnd = len(data)
	}

	options, optEnd, err := parseOptions(data[HeaderSize:headerEnd])
	if err != nil {
		return nil, nil, nil, err
	}

	payload := data[HeaderSize+optEnd : len(data)]

	return hdr, options, payload, nil
}

func parseOptions(data []byte) ([]DCCPOption, int, error) {
	var options []DCCPOption
	offset := 0
	for offset < len(data) {
		if data[offset] == byte(OptPadding) {
			offset++
			continue
		}
		if offset+1 >= len(data) {
			break
		}
		optType := OptionType(data[offset])
		optLen := int(data[offset+1])
		if optLen < 2 {
			return options, offset, ErrInvalidOption
		}
		if offset+optLen > len(data) {
			break
		}
		opt := DCCPOption{
			Type: optType,
			Len:  data[offset+1],
			Data: make([]byte, optLen-2),
		}
		copy(opt.Data, data[offset+2:offset+optLen])
		options = append(options, opt)
		offset += optLen
	}
	return options, offset, nil
}

func computeChecksum(data []byte) uint16 {
	var sum uint32
	length := len(data)
	if length%2 != 0 {
		length--
	}
	for i := 0; i < length; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}
	if len(data)%2 != 0 {
		sum += uint32(data[len(data)-1]) << 8
	}
	for sum>>16 != 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return ^uint16(sum)
}

func BuildRequestPacket(srcPort, dstPort uint16, serviceCode ServiceCode, initSeq uint64) ([]byte, error) {
	hdr := &DCCPHeader{
		SourcePort: srcPort,
		DestPort:   dstPort,
		CCVal:      0,
		Type:       PktDCCPRequest,
		Extended:   1,
	}
	hdr.SequenceNumber = initSeq

	options := []DCCPOption{
		{
			Type: OptInitCookie,
			Len:  6,
			Data: append([]byte{0, 0}, serviceCode[:]...),
		},
		{
			Type: OptFeature,
			Len:  5,
			Data: []byte{byte(CCID4), 0, 0},
		},
		{
			Type: OptTimestamp,
			Len:  10,
			Data: make([]byte, 8),
		},
	}

	return EncodeDCCPPacket(hdr, options, nil)
}

func BuildResponsePacket(srcPort, dstPort uint16, serviceCode ServiceCode, respSeq uint64, initSeq uint64) ([]byte, error) {
	hdr := &DCCPHeader{
		SourcePort: srcPort,
		DestPort:   dstPort,
		CCVal:      0,
		Type:       PktDCCPResponse,
		Extended:   1,
	}
	hdr.SequenceNumber = respSeq

	options := []DCCPOption{
		{
			Type: OptInitCookie,
			Len:  6,
			Data: append([]byte{0, 0}, serviceCode[:]...),
		},
		{
			Type: OptConfirmL,
			Len:  6,
			Data: make([]byte, 4),
		},
		{
			Type: OptTimestampEcho,
			Len:  10,
			Data: make([]byte, 8),
		},
	}

	return EncodeDCCPPacket(hdr, options, nil)
}

func BuildDataPacket(srcPort, dstPort uint16, seq uint64, payload []byte) ([]byte, error) {
	hdr := &DCCPHeader{
		SourcePort: srcPort,
		DestPort:   dstPort,
		CCVal:      0,
		Type:       PktDCCPData,
		Extended:   1,
	}
	hdr.SequenceNumber = seq

	return EncodeDCCPPacket(hdr, nil, payload)
}

func BuildAckPacket(srcPort, dstPort uint16, ackSeq uint64) ([]byte, error) {
	hdr := &DCCPHeader{
		SourcePort: srcPort,
		DestPort:   dstPort,
		CCVal:      0,
		Type:       PktDCCPAck,
		Extended:   1,
	}
	hdr.SequenceNumber = ackSeq

	return EncodeDCCPPacket(hdr, nil, nil)
}

func BuildClosePacket(srcPort, dstPort uint16, seq uint64) ([]byte, error) {
	hdr := &DCCPHeader{
		SourcePort: srcPort,
		DestPort:   dstPort,
		CCVal:      0,
		Type:       PktDCCPClose,
		Extended:   1,
	}
	hdr.SequenceNumber = seq

	return EncodeDCCPPacket(hdr, nil, nil)
}

func BuildResetPacket(srcPort, dstPort uint16, seq uint64, reason byte) ([]byte, error) {
	hdr := &DCCPHeader{
		SourcePort: srcPort,
		DestPort:   dstPort,
		CCVal:      0,
		Type:       PktDCCPReset,
		Extended:   1,
	}
	hdr.SequenceNumber = seq

	return EncodeDCCPPacket(hdr, nil, []byte{reason})
}

func ReadFullDCCPPacket(r io.Reader) (*DCCPHeader, []DCCPOption, []byte, error) {
	var hdrBuf [HeaderSize]byte
	if _, err := io.ReadFull(r, hdrBuf[:]); err != nil {
		return nil, nil, nil, err
	}

	dataOffset := ((hdrBuf[4] >> 4) & 0x0F) * 4
	needed := int(dataOffset) - HeaderSize
	if needed < 0 {
		needed = 0
	}

	readBuf := make([]byte, HeaderSize+needed)
	copy(readBuf, hdrBuf[:])

	if needed > 0 {
		if _, err := io.ReadFull(r, readBuf[HeaderSize:]); err != nil {
			return nil, nil, nil, err
		}
	}

	return DecodeDCCPPacket(readBuf)
}
