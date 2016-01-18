// markerpdu
package lacp

import ()

type LaMarkerPdu struct {
	subType                   uint8 // 2
	version                   uint8
	tlv_type_marker_info      uint8
	marker_information_length uint8 // 16
	requestor_port            uint16
	requestor_System          [6]uint8
	requestor_transaction_id  [4]uint8
	pad                       uint16
	tlv_type_terminator       uint8 // terminator 0x00
	terminator_length         uint8
	reserved                  [90]uint8
	// FCS 4 bytes
}
