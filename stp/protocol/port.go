// port.go
package stp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"strings"
	"sync"
	"time"
)

var PortMapTable map[int32]*StpPort
var PortConfigMap map[int32]portConfig

const PortConfigModuleStr = "Port Config"

type portConfig struct {
	Name         string
	HardwareAddr net.HardwareAddr
}

type StpPort struct {
	IfIndex int32

	// 17.19
	AgeingTime         int32
	Agree              bool
	Agreed             bool
	DesignatedPriority PriorityVector
	DesignatedTimes    Times
	Disputed           bool
	FdbFlush           bool
	Forward            bool
	Forwarding         bool
	Infols             PortInfoState
	Learn              bool
	Learning           bool
	Mcheck             bool
	MsgPriority        PriorityVector
	MsgTimes           Times
	NewInfo            bool
	OperEdge           bool
	PortEnabled        bool
	PortId             int32
	PortPathCost       int32
	PortPriority       PriorityVector
	PortTimes          Times
	Proposed           bool
	Proposing          bool
	RcvdBPDU           bool
	RcvdfInfo          PortDesignatedRcvInfo
	RcvdMsg            bool
	RcvdRSTP           bool
	RcvdSTP            bool
	RcvdTc             bool
	RcvdTcAck          bool
	RcvdTcn            bool
	ReRoot             bool
	Reselect           bool
	Role               PortRole
	Selected           bool
	SelectedRole       PortRole
	SendRSTP           bool
	Sync               bool
	Synced             bool
	TcAck              bool
	TcProp             bool
	Tick               bool
	TxCount            uint64
	UpdtInfo           bool

	// statistics
	BpduRx uint64

	// 17.17
	EdgeDelayWhileTimer PortTimer
	FdWhileTimer        PortTimer
	HelloWhenTimer      PortTimer
	MdelayWhiletimer    PortTimer
	RbWhileTimer        PortTimer
	RcvdInfoWhiletimer  PortTimer
	RrWhileTimer        PortTimer
	TcWhileTimer        PortTimer

	PrxmMachineFsm *PrxmMachine
	PtmMachineFsm  *PtmMachine
	PpmmMachineFsm *PpmmMachine
	PtxmMachineFsm *PtxmMachine

	begin bool

	// handle used to tx packets to linux if
	handle *pcap.Handle

	// a way to sync all machines
	wg sync.WaitGroup
	// chanel to send response messages
	portChan chan string
}

type PortTimer struct {
	count int32
}

func NewStpPort(c *StpPortConfig) *StpPort {

	/*
		MigrateTimeDefault = 3
		BridgeHelloTimeDefault = 2
		BridgeMaxAgeDefault = 20
		BridgeForwardDelayDefault = 15
		TransmitHoldCountDefault = 6
	*/

	p := &StpPort{
		IfIndex: c.Dot1dStpPortKey,
		PortId:  c.Dot1dStpPortKey,
		//PortPriority: c.Dot1dStpPortPriority,
		PortEnabled:  c.Dot1dStpPortEnable,
		PortPathCost: c.Dot1dStpPortPathCost,
		DesignatedTimes: Times{
			// TODO fill in
			ForwardingDelay: BridgeMaxAgeDefault,    // TOOD
			HelloTime:       BridgeHelloTimeDefault, // TODO
			MaxAge:          BridgeMaxAgeDefault,    // TODO
			MessageAge:      0,                      // TODO
		},
		EdgeDelayWhileTimer: PortTimer{count: MigrateTimeDefault},
		FdWhileTimer:        PortTimer{count: BridgeMaxAgeDefault}, // TODO same as ForwardingDelay above
		HelloWhenTimer:      PortTimer{count: BridgeHelloTimeDefault},
		MdelayWhiletimer:    PortTimer{count: MigrateTimeDefault},
		RbWhileTimer:        PortTimer{count: BridgeHelloTimeDefault * 2},
		RcvdInfoWhiletimer:  PortTimer{count: BridgeHelloTimeDefault * 3},
		RrWhileTimer:        PortTimer{count: BridgeMaxAgeDefault},
		TcWhileTimer:        PortTimer{count: BridgeHelloTimeDefault}, // should be updated by newTcWhile func
		portChan:            make(chan string),
	}
	// add port to map table
	if _, ok := PortMapTable[p.IfIndex]; !ok {
		PortMapTable = make(map[int32]*StpPort, 0)
	}

	PortMapTable[p.IfIndex] = p

	// lets setup the port receive/transmit handle
	ifName, _ := PortConfigMap[p.IfIndex]
	handle, err := pcap.OpenLive(ifName.Name, 65536, false, 50*time.Millisecond)
	if err != nil {
		// failure here may be ok as this may be SIM
		if !strings.Contains(ifName.Name, "SIM") {
			fmt.Println("Error creating pcap OpenLive handle for port", p.IfIndex, ifName.Name, err)
		}
		return p
	}
	fmt.Println("Creating STP Listener for intf", p.IfIndex, ifName.Name)
	//p.LaPortLog(fmt.Sprintf("Creating Listener for intf", p.IntfNum))
	p.handle = handle
	src := gopacket.NewPacketSource(p.handle, layers.LayerTypeEthernet)
	in := src.Packets()
	// start rx routine
	BpduRxMain(p.IfIndex, in)
	return p

}

func DelStpPort(p *StpPort) {
	p.Stop()
	// remove from global port table
	delete(PortMapTable, p.IfIndex)
}

func StpFindPortById(pId int32, p **StpPort) bool {
	for ifindex, port := range PortMapTable {
		if ifindex == pId {
			*p = port
			return true
		}
	}
	return false
}

func (p *StpPort) Stop() {

	// close rx/tx processing
	if p.handle != nil {
		p.handle.Close()
		fmt.Println("RX/TX handle closed for port", p.IfIndex)
	}

	if p.PrxmMachineFsm != nil {
		p.PrxmMachineFsm.Stop()
		p.PrxmMachineFsm = nil
	}

	if p.PtmMachineFsm != nil {
		p.PtmMachineFsm.Stop()
		p.PtmMachineFsm = nil
	}

	if p.PpmmMachineFsm != nil {
		p.PpmmMachineFsm.Stop()
		p.PpmmMachineFsm = nil
	}

	// lets wait for the machines to close
	p.wg.Wait()
	close(p.portChan)

}

func (p *StpPort) ResetCounter(counterType TimerType) {
	// TODO
}

func (p *StpPort) DecrementTimerCounters() {
	if p.EdgeDelayWhileTimer.count != 0 {
		p.EdgeDelayWhileTimer.count--
	}
	if p.FdWhileTimer.count != 0 {
		p.FdWhileTimer.count--
	}
	if p.HelloWhenTimer.count != 0 {
		p.HelloWhenTimer.count--
	}
	if p.MdelayWhiletimer.count != 0 {
		p.MdelayWhiletimer.count--
	}
	if p.RbWhileTimer.count != 0 {
		p.RbWhileTimer.count--
	}
	if p.RcvdInfoWhiletimer.count != 0 {
		p.RcvdInfoWhiletimer.count--
	}
	if p.RrWhileTimer.count != 0 {
		p.RrWhileTimer.count--
	}
	if p.TcWhileTimer.count != 0 {
		p.TcWhileTimer.count--
	}
}

func (p *StpPort) BEGIN(restart bool) {
	mEvtChan := make([]chan MachineEvent, 0)
	evt := make([]MachineEvent, 0)

	if !restart {

		// start all the State machines
		// will send event to Mux machine
		// thus machine must be up and
		// running first
		// Port Timer State Machine
		p.PtmMachineMain()
		// Port Receive State Machine
		p.PrxmMachineMain()
	}

	// 1) Port Receive Machine
	// 2) Port Trasmit Machine
	// Prxm
	if p.PrxmMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.PrxmMachineFsm.PrxmEvents)
		evt = append(evt, MachineEvent{e: PrxmEventBegin,
			src: PortConfigModuleStr})
	}

	// Ptm
	if p.PtmMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.PtmMachineFsm.PtmEvents)
		evt = append(evt, MachineEvent{e: PtmEventBegin,
			src: PortConfigModuleStr})
	}

	// Ppm
	if p.PpmmMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.PpmmMachineFsm.PpmmEvents)
		evt = append(evt, MachineEvent{e: PpmmEventBegin,
			src: PortConfigModuleStr})
	}
	// call the begin event for each
	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, true)

	p.begin = false
}

// DistributeMachineEvents will distribute the events in parrallel
// to each machine
func (p *StpPort) DistributeMachineEvents(mec []chan MachineEvent, e []MachineEvent, waitForResponse bool) {

	length := len(mec)
	if len(mec) != len(e) {
		fmt.Println("STPPORT: Distributing of events failed")
		return
	}

	// send all begin events to each machine in parrallel
	for j := 0; j < length; j++ {
		go func(port *StpPort, w bool, idx int, machineEventChannel []chan MachineEvent, event []MachineEvent) {
			if w {
				event[idx].responseChan = p.portChan
			}
			event[idx].src = PortConfigModuleStr
			machineEventChannel[idx] <- event[idx]
		}(p, waitForResponse, j, mec, e)
	}

	if waitForResponse {
		i := 0
		// lets wait for all the machines to respond
		for {
			select {
			case mStr := <-p.portChan:
				i++
				fmt.Println(strings.Join([]string{"STPPORT:", mStr, "response received"}, " "))
				//fmt.Println("LAPORT: Waiting for response Delayed", length, "curr", i, time.Now())
				if i >= length {
					// 10/24/15 fixed hack by sending response after Machine.ProcessEvent
					// HACK, found that port is pre-empting the State machine callback return
					// lets delay for a short period to allow for event to be received
					// and other routines to process their events
					/*
						if p.logEna {
							time.Sleep(time.Millisecond * 3)
						} else {
							time.Sleep(time.Millisecond * 1)
						}
					*/
					return
				}
			}
		}
	}
}

func (p *StpPort) BridgeProtocolVersionGet() uint8 {
	// TODO get the protocol version from the bridge
	// Below is the default
	return layers.RSTPProtocolVersion
}

func (p *StpPort) BridgeRootPortGet() int32 {
	// TODO get the bridge RootPortId
	return 10
}
