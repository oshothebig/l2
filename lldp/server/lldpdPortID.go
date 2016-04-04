package lldpServer

// port id type def
type lldpPortIDSubtype uint8

// List of valid lldp frame port id sub types
const (
	PortIDSubtypeReserved        lldpPortIDSubtype = 0
	PortIDSubtypeInterfaceAlias  lldpPortIDSubtype = 1
	PortIDSubtypePortComponent   lldpPortIDSubtype = 2
	PortIDSubtypeMACAddress      lldpPortIDSubtype = 3
	PortIDSubtypeNetworkAddress  lldpPortIDSubtype = 4
	PortIDSubtypeInterfaceName   lldpPortIDSubtype = 5
	PortIDSubtypeAgentCircuitID  lldpPortIDSubtype = 6
	PortIDSubtypeLocallyAssigned lldpPortIDSubtype = 7
)

// A LLDPPortID is a structure parsed from a port ID TLV.
type LLDPPortID struct {
	// Subtype specifies the type of identification carried out in Port ID.
	Subtype lldpPortIDSubtype

	// ID specifies raw bytes containing identification information for
	// this PortID.
	//
	// ID may carry alphanumeric data or binary data, depending upon the
	// value of Subtype, during rx or tx
	ID []byte
}
