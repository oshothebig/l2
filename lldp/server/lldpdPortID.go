package lldpServer

import (
	"io"
)

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
	SubType lldpPortIDSubtype

	// ID specifies raw bytes containing identification information for
	// this PortID.
	//
	// ID may carry alphanumeric data or binary data, depending upon the
	// value of Subtype, during rx or tx
	ID []byte
}

// Marshall chassis id information into binary form
func (c *LLDPPortID) LLDPPortIDMarshall() ([]byte, error) {
	// SubType : 1 Byte
	// ID : N Bytes
	// This is the total length of the byte object
	b := make([]byte, 1+len(c.ID))
	// Copy SubType first
	b[0] = byte(c.SubType)
	// Copy id information
	copy(b[1:], c.ID)

	return b, nil
}

// UnMarshall chassis id information from binary form to LLDPPortID
func (c *LLDPPortID) LLDPPortIDUnMarshall(b []byte) error {
	// Mandatory field of subtype should be specified
	if len(b) < 1 {
		return io.ErrUnexpectedEOF
	}
	c.SubType = lldpPortIDSubtype(b[0])
	c.ID = make([]byte, len(b[1:]))
	copy(c.ID, b[1:])

	return nil
}
