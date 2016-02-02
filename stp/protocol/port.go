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
	InfoIs             PortInfoState
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
	RstpVersion        bool
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
	// 6.4.3
	OperPointToPointMAC   bool
	AdminPoinrtToPointMAC PointToPointMac

	// Associated Bridge Id
	BridgeId BridgeId
	b        *Bridge

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
	PimMachineFsm  *PimMachine
	BdmMachineFsm  *BdmMachine
	PrsMachineFsm  *PrsMachine
	PrtMachineFsm  *PrtMachine
	TcMachineFsm   *TcMachine
	PstMachineFsm  *PstMachine

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
	var b *Bridge
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
		SendRSTP:            true, // default
		RcvdRSTP:            true, // default
		EdgeDelayWhileTimer: PortTimer{count: MigrateTimeDefault},
		FdWhileTimer:        PortTimer{count: BridgeMaxAgeDefault}, // TODO same as ForwardingDelay above
		HelloWhenTimer:      PortTimer{count: BridgeHelloTimeDefault},
		MdelayWhiletimer:    PortTimer{count: MigrateTimeDefault},
		RbWhileTimer:        PortTimer{count: BridgeHelloTimeDefault * 2},
		RcvdInfoWhiletimer:  PortTimer{count: BridgeHelloTimeDefault * 3},
		RrWhileTimer:        PortTimer{count: BridgeMaxAgeDefault},
		TcWhileTimer:        PortTimer{count: BridgeHelloTimeDefault}, // should be updated by newTcWhile func
		portChan:            make(chan string),
		BridgeId:            c.Dot1dStpBridgeId,
	}

	if StpFindBridgeById(p.BridgeId, &b) {
		p.b = b
	}

	PortMapTable[p.IfIndex] = p

	// lets setup the port receive/transmit handle
	ifName, _ := PortConfigMap[p.IfIndex]
	handle, err := pcap.OpenLive(ifName.Name, 65536, false, 50*time.Millisecond)
	if err != nil {
		// failure here may be ok as this may be SIM
		if !strings.Contains(ifName.Name, "SIM") {
			StpLogger("ERROR", fmt.Sprintf("Error creating pcap OpenLive handle for port %d %s %s\n", p.IfIndex, ifName.Name, err))
		}
		return p
	}
	StpLogger("INFO", fmt.Sprintf("Creating STP Listener for intf %d %s\n", p.IfIndex, ifName.Name))
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
		StpLogger("INFO", fmt.Sprintf("RX/TX handle closed for port %d\n", p.IfIndex))
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

	if p.PtxmMachineFsm != nil {
		p.PtxmMachineFsm.Stop()
		p.PtxmMachineFsm = nil
	}

	if p.PimMachineFsm != nil {
		p.PimMachineFsm.Stop()
		p.PimMachineFsm = nil
	}

	if p.BdmMachineFsm != nil {
		p.BdmMachineFsm.Stop()
		p.BdmMachineFsm = nil
	}

	if p.PrtMachineFsm != nil {
		p.PrtMachineFsm.Stop()
		p.PrtMachineFsm = nil
	}

	if p.TcMachineFsm != nil {
		p.TcMachineFsm.Stop()
		p.TcMachineFsm = nil
	}

	if p.PstMachineFsm != nil {
		p.PstMachineFsm.Stop()
		p.PstMachineFsm = nil
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
		// Port Timer State Machine
		p.PtmMachineMain()
		// Port Receive State Machine
		p.PrxmMachineMain()
		// Port Protocol Migration State Machine
		p.PpmmMachineMain()
		// Port Transmit State Machine
		p.PtxmMachineMain()
		// Port Information State Machine
		p.PimMachineMain()
		// Bridge Detection State Machine
		p.BdmMachineMain()
		// Port Role Selection State Machine (one instance per bridge)
		//p.PrsMachineMain()
		// Port Role Transitions State Machine
		p.PrtMachineMain()
		// Topology Change State Machine
		p.TcMachineMain()
		// Port State Transition State Machine
		p.PstMachineMain()
	}

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

	// Ppm
	if p.PtxmMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.PtxmMachineFsm.PtxmEvents)
		evt = append(evt, MachineEvent{e: PtxmEventBegin,
			src: PortConfigModuleStr})
	}

	// Pim
	if p.PimMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.PimMachineFsm.PimEvents)
		evt = append(evt, MachineEvent{e: PimEventBegin,
			src: PortConfigModuleStr})
	}

	// Bdm
	if p.BdmMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.BdmMachineFsm.BdmEvents)
		// TODO add logic to determin Admin Edge
		evt = append(evt, MachineEvent{e: BdmEventBeginAdminEdge,
			src: PortConfigModuleStr})
	}

	// Prtm
	if p.PrtMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.PrtMachineFsm.PrtEvents)
		evt = append(evt, MachineEvent{e: PrtEventBegin,
			src: PortConfigModuleStr})
	}

	// Tcm
	if p.TcMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.TcMachineFsm.TcEvents)
		evt = append(evt, MachineEvent{e: TcEventBegin,
			src: PortConfigModuleStr})
	}

	// Pstm
	if p.PstMachineFsm != nil {
		mEvtChan = append(mEvtChan, p.PstMachineFsm.PstEvents)
		evt = append(evt, MachineEvent{e: PstEventBegin,
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
		StpLogger("ERROR", "STPPORT: Distributing of events failed")
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
				StpLogger("INFO", strings.Join([]string{"STPPORT:", mStr, "response received"}, " "))
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

/*
BPDU info
ProtocolId        uint16
	ProtocolVersionId byte
	BPDUType          byte
	Flags             byte
	RootId            [8]byte
	RootCostPath      uint32
	BridgeId          [8]byte
	PortId            uint16
	MsgAge            uint16
	MaxAge            uint16
	HelloTime         uint16
	FwdDelay          uint16
*/

func (p *StpPort) SaveMsgRcvInfo(data interface{}) {

	switch data.(type) {
	case layers.STP:
		stp := data.(*layers.STP)
		// TODO revisit what the BridgePortId should be
		p.MsgPriority.BridgePortId = stp.PortId
		p.MsgPriority.DesignatedBridgeId = stp.BridgeId
		p.MsgPriority.DesignatedPortId = stp.PortId
		p.MsgPriority.RootBridgeId = stp.RootId
		p.MsgPriority.RootPathCost = stp.RootPathCost

		p.MsgTimes.ForwardingDelay = stp.FwdDelay
		p.MsgTimes.HelloTime = stp.HelloTime
		p.MsgTimes.MaxAge = stp.MaxAge
		p.MsgTimes.MessageAge = stp.MsgAge

	case layers.RSTP:
		rstp := data.(*layers.RSTP)
		// TODO revisit what the BridgePortId should be
		p.MsgPriority.BridgePortId = rstp.PortId
		p.MsgPriority.DesignatedBridgeId = rstp.BridgeId
		p.MsgPriority.DesignatedPortId = rstp.PortId
		p.MsgPriority.RootBridgeId = rstp.RootId
		p.MsgPriority.RootPathCost = rstp.RootPathCost

		p.MsgTimes.ForwardingDelay = rstp.FwdDelay
		p.MsgTimes.HelloTime = rstp.HelloTime
		p.MsgTimes.MaxAge = rstp.MaxAge
		p.MsgTimes.MessageAge = rstp.MsgAge

	case layers.PVST:
		pvst := data.(*layers.PVST)
		// TODO revisit what the BridgePortId should be
		p.MsgPriority.BridgePortId = pvst.PortId
		p.MsgPriority.DesignatedBridgeId = pvst.BridgeId
		p.MsgPriority.DesignatedPortId = pvst.PortId
		p.MsgPriority.RootBridgeId = pvst.RootId
		p.MsgPriority.RootPathCost = pvst.RootPathCost

		p.MsgTimes.ForwardingDelay = pvst.FwdDelay
		p.MsgTimes.HelloTime = pvst.HelloTime
		p.MsgTimes.MaxAge = pvst.MaxAge
		p.MsgTimes.MessageAge = pvst.MsgAge

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

func (p *StpPort) PortEnableSet(src string, val bool) {
	// The following Machines need to know about
	// changes in PortEnable State
	// 1) Port Receive
	// 2) Port Protocol Migration
	// 3) Port Information
	// 4) Bridge Detection
	if p.PortEnabled != val {
		mEvtChan := make([]chan MachineEvent, 0)
		evt := make([]MachineEvent, 0)
		p.PortEnabled = val

		// notify the state machines
		if p.EdgeDelayWhileTimer.count != MigrateTimeDefault && val == false {
			mEvtChan = append(mEvtChan, p.PrxmMachineFsm.PrxmEvents)
			evt = append(evt, MachineEvent{e: PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled,
				src: src})
		}
		if val == false {
			mEvtChan = append(mEvtChan, p.PpmmMachineFsm.PpmmEvents)
			evt = append(evt, MachineEvent{e: PpmmEventNotPortEnabled,
				src: src})
			// TODO need AdminEdge variable
			// if p.AdminEdge == false {
			//BdEventNotPortEnabledAndNotAdminEdge
			//} else {
			//BdmEventNotPortEnabledAndAdminEdge
			//}
		} else {
			mEvtChan = append(mEvtChan, p.PimMachineFsm.PimEvents)
			evt = append(evt, MachineEvent{e: PimEventPortEnabled,
				src: src})
		}
		// distribute the events
		p.DistributeMachineEvents(mEvtChan, evt, false)
	}
}

func (p *StpPort) ForceVersionSet(src string, val int) {
	// The following machines need to know about
	// changes in forceVersion State
	// 1) Port Protocol Migration
	// 2) Port Role Transitions
	// 3) Port Transmit
}

func (p *StpPort) UpdtInfoSet(src string, val bool) {
	// The following machines need to know about
	// changes in UpdtInfo State
	// 1) Port Information
	// 2) Port Role Transitions
	// 3) Port Transmit
	p.UpdtInfo = val
}

func (p *StpPort) DesignatedPrioritySet(src string, val *PriorityVector) {
	// The following machines need to know about
	// changes in DesignatedPriority State
	// 1) Port Information
	// 2) Port Transmit
	p.DesignatedPriority = *val
}

func (p *StpPort) DesignatedTimesSet(src string, val *Times) {
	// The following machines need to know about
	// changes in DesignatedTimes State
	// 1) Port Information
	// 2) Port Transmit
	p.DesignatedTimes = *val
}

func (p *StpPort) SelectedSet(src string, val bool) {
	// The following machines need to know about
	// changes in Selected State
	// 1) Port Information
	// 2) Port Role Selection
	// 3) Port Role Transitions
	p.Selected = val
}

func (p *StpPort) OperEdgeSet(src string, val bool) {
	// The following machines need to know about
	// changes in OperEdge State
	// 1) Port Role Transitions
	// 2) Bridge Detection
	p.OperEdge = val
}

func (p *StpPort) RoleSet(src string, val PortRole) {
	// The following machines need to know about
	// changes in Role State
	// 1) Port State Transitions
	// 2) Topology Change
	p.Role = val
}
