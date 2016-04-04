package lldpServer

// tlv type def
type lldpTLVType uint8

const (
	// max tlv length of value data which is allowed in TLV
	LLDPTLVLengthMax = 0x01ff

	// Mandatory TLVType values in all LLDPDUs or LLDP Frame
	// TLVTypeEnd is a special sentinel value used to indicate the end of
	// TLVs in a LLDPDU or LLDP Frame
	TLVTypeEnd       lldpTLVType = 0
	TLVTypeChassisID lldpTLVType = 1
	TLVTypePortID    lldpTLVType = 2
	TLVTypeTTL       lldpTLVType = 3

	// Optional TLVType values which may occur in LLDPDUs or LLDP Frame
	TLVTypePortDescription    lldpTLVType = 4
	TLVTypeSystemName         lldpTLVType = 5
	TLVTypeSystemDescription  lldpTLVType = 6
	TLVTypeSystemCapabilities lldpTLVType = 7
	TLVTypeManagementAddress  lldpTLVType = 8

	// TLVType which can be used
	// to carry organization-specific data in a special format.
	TLVTypeOrganizationSpecific lldpTLVType = 127

	// maximum possible value for a TLVType.
	TLVTypeMax lldpTLVType = TLVTypeOrganizationSpecific
)

// TLV structure used to carry information in an encoded format.
type LLDPTLV struct {
	// Type specifies the type of value carried in TLV.
	Type lldpTLVType

	// Length specifies the length of the value carried in TLV.
	Length uint16

	// Value specifies the raw data carried in TLV.
	Value []byte
}
