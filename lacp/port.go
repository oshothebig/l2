// port
package lacp

import (
	"time"
)

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
	portNum        int
	portPriority   int
	aggId          int
	aggSelected    uint32
	nttFlag        bool
	operIndividual int
	// TRUE - Aggregation port is operable (MAC_Operational == True)
	// FALSE - otherwise
	portEnabled bool
	portMoved   bool
	begin       bool

	// administrative values for state described in 6.4.2.3
	actorAdmin   LacpPortInfo
	actorOper    LacpPortInfo
	partnerAdmin LacpPortInfo
	partnerOper  LacpPortInfo

	// timers and configuration
	periodicTxTimerInterval   time.Duration
	waitWhileTimerTimeout     time.Duration
	currentWhileTimerTimeout  time.Duration
	actorChurnTimerInterval   time.Duration
	partnerChurnTimerInterval time.Duration

	// Tickers for interval timers
	periodicTxTimer   *time.Timer
	waitWhileTimer    *time.Timer
	currentWhileTimer *time.Timer
	actorChurnTimer   *time.Timer
	partnerChurnTimer *time.Timer

	// state machines
	rxMachineFsm *LacpRxMachine

	// Version 2
	partnerLacpPduVersionNumber int
	enableLongPduXmit           bool

	// channels
	delPortSignalChannel chan bool
}

// NewLaAggPort
// Allocate a new lag port, creating appropriate timers
func NewLaAggPort(portNum int, interval time.Duration) *LaAggPort {
	port := &LaAggPort{portNum: portNum,
		begin:                    true,
		portMoved:                false,
		waitWhileTimerTimeout:    LacpLongTimeoutTime,
		waitWhileTimer:           time.NewTimer(time.Second * (interval * 3)),
		currentWhileTimerTimeout: LacpLongTimeoutTime,
		currentWhileTimer:        time.NewTimer(time.Second * (interval * 3)),
		delPortSignalChannel:     make(chan bool)}

	port.WaitWhileTimerStop()
	port.CurrentWhileTimerStop()
	return port
}

func DelLaAggPort(p *LaAggPort) {
	p.Stop()
	// close all channels created on the port
	//close(p.pktRxChannel)
	close(p.delPortSignalChannel)
}

func (p *LaAggPort) Stop() {
	// stop the timers
	p.WaitWhileTimerStop()
	// kill the port go routine
	p.delPortSignalChannel <- true

}

func (p *LaAggPort) Start() {

	p.LacpRxMachineMain()
	/*
		go func() {

			fmt.Println("Start:              Time:", time.Now())
			for {
				select {
				case delMsg := <-p.delPortSignalChannel:
					if delMsg {
						fmt.Println("Deleting port routing", p.portNum)
						return
					}
				case msg1 := <-p.waitWhileTimer.C:
					fmt.Println("RX: timeout         Time:", msg1)
					//case msg2 := <-p.pktRxChannel:
					//	fmt.Println("Rx:", msg2, "Time:", time.Now())
					//	p.WaitWhileTimerStart()
				}
			}
		}()
	*/
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
