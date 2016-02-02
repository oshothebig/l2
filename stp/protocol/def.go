// def.go
package stp

import (
	"strconv"
	"strings"
	"utils/fsm"
)

type BPDURxType int8

// this is not to be confused with bpdu type as defined in message
const (
	BPDURxTypeUnknown BPDURxType = iota
	BPDURxTypeUnknownBPDU
	BPDURxTypeSTP
	BPDURxTypeRSTP
	BPDURxTypeTopo
	BPDURxTypePVST
)

const (
	MigrateTimeDefault        = 3
	BridgeHelloTimeMin        = 1
	BridgeHelloTimeDefault    = 2
	BridgeMaxAgeMin           = 6
	BridgeMaxAgeMax           = 40
	BridgeMaxAgeDefault       = 20
	BridgeForwardDelayMin     = 4
	BridgeForwardDelayMax     = 30
	BridgeForwardDelayDefault = 15
	TransmitHoldCountMin      = 1
	TransmitHoldCountMax      = 10
	TransmitHoldCountDefault  = 6
)

// Table 17-3 Recommended Port Path Cost Values
// provisionable range 1-200,000,000
const (
	// usage 20,000,000-200,000,000
	PortPathCostSpeedLess100Kbs = 200000000
	// usage 2,000,000-200,000,000
	PortPathCostSpeed1Mb = 20000000
	// usage 200,000-20,000,000
	PortPathCostSpeed10Mb = 2000000
	// usage 20,000-2,000,000
	PortPathCostSpeed100Mb = 200000
	// usage 2,000-200,000
	PortPathCost1Gb = 20000
	// usage 200-20,000
	PortPathCost10Gb = 2000
	// usage 20-2000
	PortPathCost100Gb = 200
	// usage 20-200
	PortPathCost1Tb = 20
	// usage 1-20
	PortPathCost10Tb = 2
)

type MachineEvent struct {
	e            fsm.Event
	src          string
	data         interface{}
	responseChan chan string
}

type StpStateEvent struct {
	// current State
	s fsm.State
	// previous State
	ps fsm.State
	// current event
	e fsm.Event
	// previous event
	pe fsm.Event

	// event src
	esrc        string
	owner       string
	strStateMap map[fsm.State]string
	logEna      bool
	logger      func(string)
}

func SendResponse(msg string, responseChan chan string) {
	responseChan <- msg
}

func (se *StpStateEvent) LoggerSet(log func(string))                 { se.logger = log }
func (se *StpStateEvent) EnableLogging(ena bool)                     { se.logEna = ena }
func (se *StpStateEvent) IsLoggerEna() bool                          { return se.logEna }
func (se *StpStateEvent) StateStrMapSet(strMap map[fsm.State]string) { se.strStateMap = strMap }
func (se *StpStateEvent) PreviousState() fsm.State                   { return se.ps }
func (se *StpStateEvent) CurrentState() fsm.State                    { return se.s }
func (se *StpStateEvent) PreviousEvent() fsm.Event                   { return se.pe }
func (se *StpStateEvent) CurrentEvent() fsm.Event                    { return se.e }
func (se *StpStateEvent) SetEvent(es string, e fsm.Event) {
	se.esrc = es
	se.pe = se.e
	se.e = e
}
func (se *StpStateEvent) SetState(s fsm.State) {
	se.ps = se.s
	se.s = s
	//if se.IsLoggerEna() && se.ps != se.s {
	se.logger((strings.Join([]string{"Src", se.esrc, "OldState", se.strStateMap[se.ps], "Evt", strconv.Itoa(int(se.e)), "NewState", se.strStateMap[s]}, ":")))
	//}
}

func StpSetBpduFlags(topochangeack uint8, agreement uint8, forwarding uint8, learning uint8, role PortRole, proposal uint8, topochange uint8, flags *uint8) {

	*flags |= topochangeack << 7
	*flags |= agreement << 6
	*flags |= forwarding << 5
	*flags |= learning << 4
	*flags |= uint8(role << 2)
	*flags |= proposal << 1
	*flags |= topochange << 0

}

func StpGetBpduRole(flags uint8) PortRole {
	return PortRole(flags >> 2 & 0x3)
}

func StpGetBpduProposal(flags uint8) bool {
	return flags>>1&0x1 == 1
}

func StpGetBpduTopoChangeAck(flags uint8) bool {
	return flags>>7&0x1 == 1
}

func StpGetBpduTopoChange(flags uint8) bool {
	return flags>>0&0x1 == 1
}

func StpGetBpduLearning(flags uint8) bool {
	return flags>>4&0x1 == 1
}

func StpGetBpduAgreement(flags uint8) bool {
	return flags>>6&0x1 == 1
}
