package dccp

const (
	DefaultDCCPPort    = 33445
	ProtocolName       = "dccp"
	ProtocolVersion    = 1
	MaxPacketSize      = 1500
	HeaderSize         = 12
	OptionsMaxSize     = 40
	SequenceWindowSize = 256
)

type PacketType byte

const (
	PktDCCPRequest  PacketType = 0
	PktDCCPResponse PacketType = 1
	PktDCCPData     PacketType = 2
	PktDCCPAck      PacketType = 3
	PktDCCPDataAck  PacketType = 4
	PktDCCPCloseReq PacketType = 5
	PktDCCPClose    PacketType = 6
	PktDCCPReset    PacketType = 7
	PktDCCPSync     PacketType = 8
	PktDCCPSyncAck  PacketType = 9
)

type CCID byte

const (
	CCID2 CCID = 2
	CCID3 CCID = 3
	CCID4 CCID = 4
)

type OptionType byte

const (
	OptPadding         OptionType = 0
	OptMandatory       OptionType = 1
	OptSlowReceiver    OptionType = 2
	OptChangeL         OptionType = 32
	OptConfirmL        OptionType = 33
	OptChangeR         OptionType = 34
	OptConfirmR        OptionType = 35
	OptAckVector       OptionType = 37
	OptAckVectorNonce  OptionType = 38
	OptDataDropped     OptionType = 39
	OptTimestamp       OptionType = 40
	OptTimestampEcho   OptionType = 41
	OptElapsedTime     OptionType = 42
	OptDataChecksum    OptionType = 43
	OptMinCSUM         OptionType = 44
	OptInitCookie      OptionType = 45
	OptFeature         OptionType = 46
	OptFeatureAck      OptionType = 47
)

type ServiceCode [4]byte

var (
	ServiceCodeV2Ray   ServiceCode
	ServiceCodeXRay    ServiceCode
	ServiceCodeSingBox ServiceCode
)

func init() {
	copy(ServiceCodeV2Ray[:], "V2RY")
	copy(ServiceCodeXRay[:], "XRAY")
	copy(ServiceCodeSingBox[:], "SBOX")
}

type DCCPHeader struct {
	SourcePort      uint16
	DestPort        uint16
	DataOffset      byte
	CCVal           byte
	ChecksumCoverage uint16
	Checksum        uint16
	ResType         byte
	Type            PacketType
	Extended        byte
	SequenceNumber  uint64
}

func (h *DCCPHeader) SetExtendedSequence(seq uint64) {
	h.Extended = 1
	h.SequenceNumber = seq
}

func (h *DCCPHeader) GetSequenceNumber() uint64 {
	return h.SequenceNumber
}
