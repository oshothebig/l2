// port
package lacp

import (
	"strings"
	"utils/fsm"
)

type PortProperties struct {
	speed  int
	duplex int
	mtu    int
}

type LacpPortSystemInfo struct {
	systemPriority uint16
	systemId       [6]uint8
}

type LacpPortInfo struct {
	system   LacpPortSystemInfo
	key      uint16
	port_pri uint16
	port     uint16
	state    uint8
}

// 802.1ax Section 6.4.7
// Port attributes associated with aggregator
type LaAggPort struct {
	// 802.1ax-2014 Section 6.3.4:
	// Link Aggregation Control uses a Port Identifier (Port ID), comprising
	// the concatenation of a Port Priority (7.3.2.1.15) and a Port Number
	// (7.3.2.1.14), to identify the Aggregation Port....
	// The most significant and second most significant octets are the first
	// and second most significant octets of the Port Priority, respectively.
	// The third and fourth most significant octets are the first and second
	// most significant octets of the Port Number, respectively.
	portId int

	// string id of port
	intfNum string

	// used to form portId
	portNum      int
	portPriority int

	aggId int
	// Once selected reference to agg group will be made
	aggAttached *LaAggregator
	aggSelected uint32
	// unable to aggregate with other links in an agg
	operIndividual int
	lacpEnabled    bool
	// TRUE - Aggregation port is operable (MAC_Operational == True)
	// FALSE - otherwise
	portEnabled  bool
	portMoved    bool
	begin        bool
	actorChurn   bool
	partnerChurn bool
	readyN       bool

	macProperties PortProperties

	// administrative values for state described in 6.4.2.3
	actorAdmin   LacpPortInfo
	actorOper    LacpPortInfo
	partnerAdmin LacpPortInfo
	partnerOper  LacpPortInfo

	// state machines
	rxMachineFsm  *LacpRxMachine
	ptxMachineFsm *LacpPtxMachine
	txMachineFsm  *LacpTxMachine
	cdMachineFsm  *LacpCdMachine
	muxMachineFsm *LacpMuxMachine

	// will serialize state transition logging per port
	LacpDebug *LacpDebug

	beginChan chan string
	log       chan string
	logEna    bool

	// Version 2
	partnerLacpPduVersionNumber int
	enableLongPduXmit           bool
	// packet is 1 byte, but spec says save as int.
	// going to save as byte
	partnerVersion uint8
}

// global port map representation of the LaAggPorts
var portMap = make(map[int]*LaAggPort)

func (p *LaAggPort) LaPortLog(msg string) {
	if p.logEna {
		p.log <- msg
	}
}

// find a port from the global map table
func LaFindPortById(pId int, p *LaAggPort) bool {
	p, ok := portMap[pId]
	return ok
}

// NewLaAggPort
// Allocate a new lag port, creating appropriate timers
func NewLaAggPort(portNum int, intfNum string) *LaAggPort {
	port := &LaAggPort{portNum: portNum,
		intfNum:     intfNum,
		begin:       true,
		portMoved:   false,
		lacpEnabled: false,
		portEnabled: false,
		beginChan:   make(chan string)}

	// add port to port map
	portMap[portNum] = port

	return port
}

func DelLaAggPort(p *LaAggPort) {
	p.Stop()
	// remove the port from the port map
	delete(portMap, p.portNum)
}

func (p *LaAggPort) Stop() {
	// stop the state machines
	p.rxMachineFsm.Stop()
	p.ptxMachineFsm.Stop()
	p.txMachineFsm.Stop()
	p.cdMachineFsm.Stop()
	p.muxMachineFsm.Stop()
	close(p.beginChan)
}

type MachineEventChanAndBegingEvent struct {
	mEvt chan string
	evt  int
}

//  Start will initiate all the state machines
// and will send an event back to this caller
// to begin processing.
func (p *LaAggPort) Start(restart bool) {
	mEvtChan := make([]chan fsm.Event, 0)
	evt := make([]fsm.Event, 0)

	// system in being initalized
	p.begin = true

	if !restart {
		// MUST BE CALLED FIRST!!!
		p.LacpDebugEventLogMain()

		// save off a shortcut to the log chan
		p.log = p.LacpDebug.LacpLogChan
		p.logEna = true

		// start all the state machines
		p.LacpRxMachineMain()

		// No begin event
		p.LacpTxMachineMain()

		p.LacpPtxMachineMain()

		p.LacpCdMachineMain()

		p.LacpMuxMachineMain()
	}
	// Rxm
	mEvtChan = append(mEvtChan, p.rxMachineFsm.RxmEvents)
	evt = append(evt, LacpRxmEventBegin)
	// Ptxm
	mEvtChan = append(mEvtChan, p.ptxMachineFsm.PtxmEvents)
	evt = append(evt, LacpPtxmEventBegin)
	// Cdm
	mEvtChan = append(mEvtChan, p.cdMachineFsm.CdmEvents)
	evt = append(evt, LacpCdmEventBegin)
	// Muxm
	mEvtChan = append(mEvtChan, p.muxMachineFsm.MuxmEvents)
	evt = append(evt, LacpMuxmEventBegin)

	// call the begin event for each
	for j := 0; j < len(mEvtChan); j++ {
		go func(idx int, evtChannel []chan fsm.Event) {
			evtChannel[idx] <- evt[idx]
		}(j, mEvtChan)
	}

	// lets wait for all the machines to respond
	for i := 0; i < len(mEvtChan); i++ {
		select {
		case mStr := <-p.beginChan:
			p.LaPortLog(strings.Join([]string{"LAPORT:", mStr, "running"}, " "))
		}
	}

	p.begin = false
}

// LacpCopyLacpPortInfo:
// Copy the LacpPortInfo data from->to
func LacpCopyLacpPortInfo(fromPortInfoPtr *LacpPortInfo, toPortInfoPtr *LacpPortInfo) {
	toPortInfoPtr.key = fromPortInfoPtr.key
	toPortInfoPtr.port = fromPortInfoPtr.port
	toPortInfoPtr.port_pri = fromPortInfoPtr.port_pri
	toPortInfoPtr.state = fromPortInfoPtr.state
	toPortInfoPtr.system.systemId = fromPortInfoPtr.system.systemId
	toPortInfoPtr.system.systemPriority = fromPortInfoPtr.system.systemPriority
}

// LacpLacpPortInfoIsEqual:
// Compare the LacpPortInfo data except be selective
// about the state bits that is being compared against
func LacpLacpPortInfoIsEqual(aPortInfoPtr *LacpPortInfo, bPortInfoPtr *LacpPortInfo, stateBits uint8) bool {

	return aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[0] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[1] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[2] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[3] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[4] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[5] &&
		aPortInfoPtr.system.systemPriority == bPortInfoPtr.system.systemPriority &&
		aPortInfoPtr.port == bPortInfoPtr.port &&
		aPortInfoPtr.port_pri == bPortInfoPtr.port_pri &&
		aPortInfoPtr.key == bPortInfoPtr.key &&
		(LacpStateIsSet(aPortInfoPtr.state, stateBits) && LacpStateIsSet(bPortInfoPtr.state, stateBits))
}
