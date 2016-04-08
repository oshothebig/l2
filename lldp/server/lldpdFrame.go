package lldpServer

import (
	"encoding/binary"
	"errors"
	"time"
)

// A LLDPFrame is an lldp frame or lldp Data Unit aka LLDPDU. A frame carries
// infomration related to device in a series of TLV (type-length-value) structs
type LLDPFrame struct {
	// ChassisID will indicate Chassis ID information regarding a device. This
	// field is mandatory.
	ChassisID *LLDPChassisID

	// PortID will indicate Port ID information regarding a device. This
	// field is mandatory
	PortID *LLDPPortID

	// TTL will indicate how long the info in the frame should be considered
	// valid. This field is mandatory
	TTL time.Duration

	// Optional specifies zero or more optional TLV values in raw format.
	// This is also mendatory
	Optional []*LLDPTLV
}

var (
	LLDP_ERR_INVALID_FRAME           = errors.New("invalid frame")
	LLDP_ERR_INCOMPLETE_FRAME        = errors.New("LLDP Frame has in-complete information")
	LLDP_ERR_INVALID_CHASSIS_ID_INFO = errors.New("invalid Chassis id info")
	LLDP_ERR_INVALID_PORT_ID_INFO    = errors.New("invalid port id info")
	LLDP_ERR_INVALID_TTL_INFO        = errors.New("invalid ttl info")
	LLDP_ERR_INVALID_END_INFO        = errors.New("end of lldp frame is not defined")
)

const (
	LLDP_MAX_TTL_VALUE      = 65535
	LLDP_MANDATORY_TLV_SIZE = 4
	LLDP_TTL_BYTE_VALUE     = 2
)

// Marshall LLDP Frame and return byte
func (obj *LLDPFrame) LLDPFrameMarshall() ([]byte, error) {
	// Check whether ChassisID is valid or not
	if obj.ChassisID == nil {
		return nil, LLDP_ERR_INVALID_FRAME
	}
	// check whether PortID is valid or not
	if obj.PortID == nil {
		return nil, LLDP_ERR_INVALID_FRAME
	}

	// Check TTL value
	timeTTL := obj.TTL / time.Second
	if timeTTL > LLDP_MAX_TTL_VALUE {
		return nil, LLDP_ERR_INVALID_FRAME
	}
	ttl := uint16(timeTTL)
	b := make([]byte, obj.MinLength())

	// Add chassis id Info
	chassisByte, _ := obj.ChassisID.LLDPChassisIDMarshall()
	chassisTLV := &LLDPTLV{
		Type:   TLVTypeChassisID,
		Length: uint16(len(chassisByte)),
		Value:  chassisByte,
	}
	offset := 0
	err := chassisTLV.LLDPTLVMarshallInsert(b, &offset)
	if err != nil {
		return nil, err
	}

	// Add Port id Info
	portByte, _ := obj.PortID.LLDPPortIDMarshall()
	portTLV := &LLDPTLV{
		Type:   TLVTypePortID,
		Length: uint16(len(portByte)),
		Value:  portByte,
	}
	err = portTLV.LLDPTLVMarshallInsert(b, &offset)
	if err != nil {
		return nil, err
	}

	// Add TTL info
	ttlTLV := &LLDPTLV{
		Type:   TLVTypeTTL,
		Length: LLDP_TTL_BYTE_VALUE,
	}
	binary.BigEndian.PutUint16(ttlTLV.Value, ttl)
	err = ttlTLV.LLDPTLVMarshallInsert(b, &offset)
	if err != nil {
		return nil, err
	}

	// Add optional tlv's if any
	for _, optlv := range obj.Optional {
		err = optlv.LLDPTLVMarshallInsert(b, &offset)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

// UnMarshall incoming byte and form a frame
func (obj *LLDPFrame) LLDPFrameUnMarshall(b []byte) (err error) {
	var tlvInfo []*LLDPTLV
	// Calculate all TLV's
	for idx := 0; len(b[idx:]) > 0; {
		// Unmarshalling one TLV at a time
		temp := new(LLDPTLV)
		if err = temp.LLDPTLVUnmarshalBinary(b[idx:]); err != nil {
			return err
		}

		// Next TLV....
		idx += 2 + int(temp.Length)
		tlvInfo = append(tlvInfo, temp)
	}
	// Get the length of all tlv's...
	tlvLen := len(tlvInfo)

	// check whether four mandatory tlv's are present or not
	if tlvLen < LLDP_MANDATORY_TLV_SIZE {
		return LLDP_ERR_INCOMPLETE_FRAME
	}

	/*  Check tlvInfo bytes before creating a frame:
	 *     [0] chassis id
	 *     [1] port id
	 *     [2] ttl
	 *     [3]...[N] end of lldpdu or lldp frame
	 */
	if tlvInfo[0].Type != TLVTypeChassisID {
		return LLDP_ERR_INVALID_CHASSIS_ID_INFO
	}
	if tlvInfo[1].Type != TLVTypePortID {
		return LLDP_ERR_INVALID_PORT_ID_INFO
	}
	if tlvInfo[2].Type != TLVTypeTTL ||
		tlvInfo[2].Length != LLDP_TTL_BYTE_VALUE {
		return LLDP_ERR_INVALID_TTL_INFO
	}
	if tlvInfo[tlvLen-1].Type != TLVTypeEnd || tlvInfo[tlvLen-1].Length != 0 {
		return LLDP_ERR_INVALID_END_INFO
	}

	// Get Chassis Id Info
	obj.ChassisID = new(LLDPChassisID)
	if err = obj.ChassisID.LLDPChassisIDUnMarshall(tlvInfo[0].Value); err != nil {
		return err
	}

	// Get Port Id Info
	obj.PortID = new(LLDPPortID)
	if err = obj.PortID.LLDPPortIDUnMarshall(tlvInfo[1].Value); err != nil {
		return err
	}

	// Get TTL Info
	obj.TTL = time.Duration(binary.BigEndian.Uint16(tlvInfo[2].Value)) * time.Second

	// Get Optional/Remaining TLV Info
	obj.Optional = tlvInfo[3:(tlvLen - 1)]

	return nil
}

// Calculate minimum mandatory lldp frame length
func (obj *LLDPFrame) MinLength() int {
	var length int
	length += 2 + 1 + len(obj.ChassisID.ID)
	length += 2 + 1 + len(obj.PortID.ID)
	length += 2 + 2

	// Optional TLV length calculation
	for _, tlv := range obj.Optional {
		length += 2 + len(tlv.Value)
	}

	// LLDP Frame or LLDPDU end of frame
	length += 2

	return length
}
