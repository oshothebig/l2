// port.go
package stp

import (
	"asicd/pluginManager/pluginCommon"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/vishvananda/netlink"
	"net"
	"strings"
	"sync"
	//"syscall"
	"time"
)

var PortMapTable map[int32]*StpPort
var PortListTable []*StpPort
var PortConfigMap map[int32]portConfig

const PortConfigModuleStr = "Port Config"

type portConfig struct {
	Name         string
	HardwareAddr net.HardwareAddr
	Speed        int32
	PortNum      int32
	IfIndex      int32
}

type StpPort struct {
	IfIndex        int32
	ProtocolPortId uint16

	// 17.19
	AgeingTime   int32
	Agree        bool
	Agreed       bool
	AdminEdge    bool
	AutoEdgePort bool // optional
	//DesignatedPriority PriorityVector
	//DesignatedTimes    Times
	Disputed     bool
	FdbFlush     bool
	Forward      bool
	Forwarding   bool
	InfoIs       PortInfoState
	Learn        bool
	Learning     bool
	Mcheck       bool
	MsgPriority  PriorityVector
	MsgTimes     Times
	NewInfo      bool
	OperEdge     bool
	PortEnabled  bool
	PortId       uint16
	PortPathCost uint32
	PortPriority PriorityVector
	PortTimes    Times
	Priority     uint16
	Proposed     bool
	Proposing    bool
	RcvdBPDU     bool
	RcvdInfo     PortDesignatedRcvInfo
	RcvdMsg      bool
	RcvdRSTP     bool
	RcvdSTP      bool
	RcvdTc       bool
	RcvdTcAck    bool
	RcvdTcn      bool
	RstpVersion  bool
	ReRoot       bool
	Reselect     bool
	Role         PortRole
	Selected     bool
	SelectedRole PortRole
	SendRSTP     bool
	Sync         bool
	Synced       bool
	TcAck        bool
	TcProp       bool
	Tick         bool
	TxCount      uint64
	UpdtInfo     bool
	// 6.4.3
	OperPointToPointMAC  bool
	AdminPointToPointMAC PointToPointMac

	// Associated Bridge Id
	BridgeId   BridgeId
	BrgIfIndex int32
	b          *Bridge

	// statistics
	BpduRx uint64
	BpduTx uint64
	StpRx  uint64
	StpTx  uint64
	TcRx   uint64
	TcTx   uint64
	RstpRx uint64
	RstpTx uint64
	PvstRx uint64
	PvstTx uint64

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
	enabled := c.Dot1dStpPortEnable
	if enabled {
		// TODO get the status from asicd
		netif, err := netlink.LinkByName(PortConfigMap[c.Dot1dStpPort].Name)
		StpLogger("INFO", fmt.Sprintf("LinkByName err %#v", err))

		if err == nil {
			netifattr := netif.Attrs()
			//if netifattr.Flags&syscall.IFF_RUNNING == syscall.IFF_RUNNING {
			if (netifattr.Flags & 1) == 1 {
				enabled = true
			}
		}
	} else {
		// in the case of tests we may not find the actual link so lets force
		// enabled to configured value
		StpLogger("INFO", fmt.Sprintf("Did not find port, forcing enabled to %t", c.Dot1dStpPortEnable))
		enabled = c.Dot1dStpPortEnable
	}

	var RootTimes Times
	if StpFindBridgeByIfIndex(c.Dot1dStpBridgeIfIndex, &b) {
		RootTimes = b.RootTimes
	}
	p := &StpPort{
		IfIndex:              c.Dot1dStpPort,
		AdminPointToPointMAC: PointToPointMac(c.Dot1dStpPortAdminPointToPoint),
		// protocol portId
		PortId:              uint16(pluginCommon.GetIdFromIfIndex(c.Dot1dStpPort)),
		Priority:            c.Dot1dStpPortPriority, // default usually 0x80
		PortEnabled:         enabled,
		PortPathCost:        uint32(c.Dot1dStpPortPathCost),
		Role:                PortRoleDisabledPort,
		PortTimes:           RootTimes,
		SendRSTP:            true,  // default
		RcvdRSTP:            false, // default
		EdgeDelayWhileTimer: PortTimer{count: MigrateTimeDefault},
		FdWhileTimer:        PortTimer{count: BridgeMaxAgeDefault}, // TODO same as ForwardingDelay above
		HelloWhenTimer:      PortTimer{count: BridgeHelloTimeDefault},
		MdelayWhiletimer:    PortTimer{count: MigrateTimeDefault},
		RbWhileTimer:        PortTimer{count: BridgeHelloTimeDefault * 2},
		RcvdInfoWhiletimer:  PortTimer{count: BridgeHelloTimeDefault * 3},
		RrWhileTimer:        PortTimer{count: BridgeMaxAgeDefault},
		TcWhileTimer:        PortTimer{count: BridgeHelloTimeDefault}, // should be updated by newTcWhile func
		portChan:            make(chan string),
		BrgIfIndex:          c.Dot1dStpBridgeIfIndex,
		AdminEdge:           c.Dot1dStpPortAdminEdgePort,
		PortPriority: PriorityVector{
			RootBridgeId:       b.BridgeIdentifier,
			RootPathCost:       0,
			DesignatedBridgeId: b.BridgeIdentifier,
			DesignatedPortId:   uint16(uint16(pluginCommon.GetIdFromIfIndex(c.Dot1dStpPort)) | c.Dot1dStpPortPriority<<8),
		},
		b: b, // reference to brige
	}

	if c.Dot1dStpPortAdminPathCost == 0 {
		// TODO need to get speed of port to automatically the port path cost
		// Table 17-3
		AutoPathCostDefaultMap := map[int32]uint32{
			10:         200000000,
			100:        20000000,
			1000:       2000000,
			10000:      200000,
			100000:     20000,
			1000000:    2000,
			10000000:   200,
			100000000:  20,
			1000000000: 2,
		}
		speed := PortConfigMap[p.IfIndex].Speed

		p.PortPathCost = AutoPathCostDefaultMap[speed]
		StpLogger("INFO", fmt.Sprintf("Auto Port Path Cost for port %d speed %d = %d", p.IfIndex, speed, p.PortPathCost))
	}

	PortMapTable[p.IfIndex] = p
	if len(PortListTable) == 0 {
		PortListTable = make([]*StpPort, 0)
	}
	PortListTable = append(PortListTable, p)

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
	StpLogger("INFO", fmt.Sprintf("NEW PORT: %#v\n", p))

	if strings.Contains(ifName.Name, "eth") {
		p.PollLinuxLinkStatus()
	}

	return p

}

func (p *StpPort) PollLinuxLinkStatus() {

	var PollingTimer *time.Timer = time.NewTimer(time.Second * 1)

	go func(p *StpPort, t *time.Timer) {
		StpMachineLogger("INFO", "LINUX POLLING", p.IfIndex, "Start")
		for {
			select {
			case <-t.C:
				// TODO get link status from asicd
				netif, _ := netlink.LinkByName(PortConfigMap[p.IfIndex].Name)
				netifattr := netif.Attrs()
				//StpLogger("INFO", fmt.Sprintf("Polling link flags%#v, running=0x%x up=0x%x check1 %t check2 %t", netifattr.Flags, syscall.IFF_RUNNING, syscall.IFF_UP, ((netifattr.Flags>>6)&0x1) == 1, (netifattr.Flags&1) == 1))
				//if (((netifattr.Flags >> 6) & 0x1) == 1) && (netifattr.Flags&1) == 1 {
				if (netifattr.Flags & 1) == 1 {
					//StpLogger("INFO", "LINUX LINK UP")
					prevPortEnabled := p.PortEnabled
					p.PortEnabled = true
					p.NotifyPortEnabled("LINUX LINK STATUS", prevPortEnabled, true)

				} else {
					//StpLogger("INFO", "LINUX LINK DOWN")
					prevPortEnabled := p.PortEnabled
					p.PortEnabled = false
					p.NotifyPortEnabled("LINUX LINK STATUS", prevPortEnabled, false)

				}
				t.Reset(time.Second * 1)
			}
		}
	}(p, PollingTimer)
}
func DelStpPort(p *StpPort) {
	p.Stop()
	// remove from global port table
	delete(PortMapTable, p.IfIndex)
	for i, delPort := range PortListTable {
		if delPort.IfIndex == p.IfIndex {
			if len(PortListTable) == 1 {
				PortListTable = nil
			} else {
				PortListTable = append(PortListTable[:i], PortListTable[i+1:]...)
			}
		}
	}
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
		if p.AdminEdge {
			evt = append(evt, MachineEvent{e: BdmEventBeginAdminEdge,
				src: PortConfigModuleStr})
		} else {
			evt = append(evt, MachineEvent{e: BdmEventBeginNotAdminEdge,
				src: PortConfigModuleStr})
		}
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

	if p.PrxmMachineFsm != nil {
		// start rx routine
		src := gopacket.NewPacketSource(p.handle, layers.LayerTypeEthernet)
		in := src.Packets()
		BpduRxMain(p.IfIndex, in)
	}
	// lets start the tick timer
	if p.PtmMachineFsm != nil {
		p.PtmMachineFsm.TickTimerStart()
	}

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

func (p *StpPort) SetRxPortCounters(ptype BPDURxType) {
	p.BpduRx++
	switch ptype {
	case BPDURxTypeSTP:
		p.StpRx++
	case BPDURxTypeRSTP:
		p.RstpRx++
	case BPDURxTypeTopo:
		p.TcRx++
	case BPDURxTypePVST:
		p.PvstRx++
	}
}

func (p *StpPort) SetTxPortCounters(ptype BPDURxType) {
	p.BpduTx++
	switch ptype {
	case BPDURxTypeSTP:
		p.StpTx++
	case BPDURxTypeRSTP:
		p.RstpTx++
	case BPDURxTypeTopo:
		p.TcTx++
	case BPDURxTypePVST:
		p.PvstTx++
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

func (p *StpPort) NotifyPortEnabled(src string, oldportenabled bool, newportenabled bool) {
	// The following Machines need to know about
	// changes in PortEnable State
	// 1) Port Receive
	// 2) Port Protocol Migration
	// 3) Port Information
	// 4) Bridge Detection
	if oldportenabled != newportenabled {
		mEvtChan := make([]chan MachineEvent, 0)
		evt := make([]MachineEvent, 0)

		// notify the state machines
		if p.EdgeDelayWhileTimer.count != MigrateTimeDefault && newportenabled == false {
			mEvtChan = append(mEvtChan, p.PrxmMachineFsm.PrxmEvents)
			evt = append(evt, MachineEvent{e: PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled,
				src: src})
		}
		if newportenabled == false {
			mEvtChan = append(mEvtChan, p.PpmmMachineFsm.PpmmEvents)
			evt = append(evt, MachineEvent{e: PpmmEventNotPortEnabled,
				src: src})

			if p.AdminEdge == false {
				//BdEventNotPortEnabledAndNotAdminEdge
				mEvtChan = append(mEvtChan, p.BdmMachineFsm.BdmEvents)
				evt = append(evt, MachineEvent{e: BdmEventNotPortEnabledAndNotAdminEdge,
					src: src})
			} else {
				//BdmEventNotPortEnabledAndAdminEdge
				mEvtChan = append(mEvtChan, p.BdmMachineFsm.BdmEvents)
				evt = append(evt, MachineEvent{e: BdmEventNotPortEnabledAndAdminEdge,
					src: src})
			}
		} else {
			mEvtChan = append(mEvtChan, p.PimMachineFsm.PimEvents)
			evt = append(evt, MachineEvent{e: PimEventPortEnabled,
				src: src})
		}
		// distribute the events
		p.DistributeMachineEvents(mEvtChan, evt, false)
	}
}

func (p *StpPort) NotifyRcvdMsgChanged(src string, oldrcvdmsg bool, newrcvdmsg bool, data interface{}) {
	// The following machines need to know about
	// changed in rcvdMsg state
	// 1) Port Receive
	// 2) Port Information
	if oldrcvdmsg != newrcvdmsg {
		if src != PrxmMachineModuleStr {
			if p.PrxmMachineFsm.Machine.Curr.CurrentState() == PrxmStateReceive &&
				p.RcvdBPDU &&
				p.PortEnabled &&
				!p.RcvdMsg {
				p.PrxmMachineFsm.PrxmEvents <- MachineEvent{
					e:   PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg,
					src: src,
				}
			}
		}
		if src != PimMachineModuleStr {
			bpdumsg := data.(RxBpduPdu)
			bpduLayer := bpdumsg.pdu

			if p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateDisabled {
				if p.RcvdMsg {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:    PimEventRcvdMsg,
						src:  src,
						data: bpduLayer,
					}
				}
			} else if p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateCurrent {
				if p.RcvdMsg &&
					!p.UpdtInfo {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:    PimEventRcvdMsgAndNotUpdtInfo,
						src:  src,
						data: bpduLayer,
					}
				} else if p.InfoIs == PortInfoStateReceived &&
					p.RcvdInfoWhiletimer.count == 0 &&
					!p.UpdtInfo &&
					!p.RcvdMsg {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:    PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg,
						src:  src,
						data: bpduLayer,
					}
				}
			}
		}
	}
}

func (p *StpPort) ForceVersionSet(src string, val int) {
	// The following machines need to know about
	// changes in forceVersion State
	// 1) Port Protocol Migration
	// 2) Port Role Transitions
	// 3) Port Transmit
}

func (p *StpPort) NotifyUpdtInfoChanged(src string, oldupdtinfo bool, newupdtinfo bool) {
	// The following machines need to know about
	// changes in UpdtInfo State
	// 1) Port Information
	// 2) Port Role Transitions
	// 3) Port Transmit
	if oldupdtinfo != newupdtinfo {
		StpMachineLogger("INFO", src, p.IfIndex, fmt.Sprintf("updateinfo changed %d", newupdtinfo))
		// PI
		if p.UpdtInfo {
			if src != PimMachineModuleStr &&
				(p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateAged ||
					p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateCurrent) {
				if p.Selected {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:   PimEventSelectedAndUpdtInfo,
						src: src,
					}
				}
			}
		} else {
			if src != PimMachineModuleStr &&
				p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateCurrent {
				if p.RcvdMsg {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:   PimEventRcvdMsgAndNotUpdtInfo,
						src: src,
					}
				} else if p.InfoIs == PortInfoStateReceived &&
					p.RcvdInfoWhiletimer.count == 0 &&
					!p.RcvdMsg {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:   PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg,
						src: src,
					}
				}
			}
			/*
				if src != PtxmMachineModuleStr &&
					p.PtxmMachineFsm.Machine.Curr.CurrentState() == PtxmStateIdle &&
					p.Selected {
					if p.HelloWhenTimer.count == 0 {
						p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
							e:   PtxmEventHelloWhenEqualsZeroAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.SendRSTP &&
						p.NewInfo &&
						p.TxCount < p.b.TxHoldCount &&
						p.HelloWhenTimer.count != 0 {
						p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
							e:   PtxmEventSendRSTPAndNewInfoAndTxCountLessThanTxHoldCoundAndHelloWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if !p.SendRSTP &&
						p.NewInfo &&
						p.Role == PortRoleRootPort &&
						p.TxCount < p.b.TxHoldCount &&
						p.HelloWhenTimer.count != 0 {
						p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
							e:   PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if !p.SendRSTP &&
						p.NewInfo &&
						p.Role == PortRoleDesignatedPort &&
						p.TxCount < p.b.TxHoldCount &&
						p.HelloWhenTimer.count != 0 {
						p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
							e:   PtxmEventNotSendRSTPAndNewInfoAndDesignatedPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}

				}
				if src != PrtMachineModuleStr &&
					p.Selected {
					if p.SelectedRole == PortRoleDisabledPort &&
						p.Role != p.SelectedRole {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSelectedRoleEqualDisabledPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
					if p.SelectedRole == PortRoleRootPort &&
						p.Role != p.SelectedRole {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSelectedRoleEqualRootPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
					if p.SelectedRole == PortRoleDesignatedPort &&
						p.Role != p.SelectedRole {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSelectedRoleEqualDesignatedPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
					if p.SelectedRole == PortRoleAlternatePort &&
						p.Role != p.SelectedRole {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSelectedRoleEqualAlternateAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
					if p.SelectedRole == PortRoleBackupPort &&
						p.Role != p.SelectedRole {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSelectedRoleEqualBackupPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
					if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDisablePort &&
						!p.Learning &&
						!p.Forwarding {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventNotLearningAndNotForwardingAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
					if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDisabledPort {
						if p.FdWhileTimer.count != int32(p.PortTimes.MaxAge) {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventFdWhileNotEqualMaxAgeAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Sync {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventSyncAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.ReRoot {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventReRootAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if !p.Synced {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventNotSyncedAndSelectedAndNotUpdtInfo,
								src: src,
							}
						}
					} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateRootPort {
						if p.Proposed &&
							!p.Agree {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.b.AllSynced() &&
							!p.Agree {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Proposed &&
							p.Agree {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if !p.Forward &&
							!p.ReRoot {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventNotForwardAndNotReRootAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.SelectedRole == PortRoleRootPort &&
							p.Role != p.SelectedRole {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventSelectedRoleEqualRootPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.RrWhileTimer.count != int32(p.PortTimes.ForwardingDelay) {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventRrWhileNotEqualFwdDelayAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.ReRoot &&
							p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventReRootAndForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.FdWhileTimer.count == 0 &&
							p.RstpVersion &&
							!p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventFdWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.ReRoot &&
							p.RbWhileTimer.count == 0 &&
							p.RstpVersion &&
							!p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventReRootedAndRbWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.FdWhileTimer.count == 0 &&
							p.RstpVersion &&
							p.Learn &&
							!p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventFdWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.ReRoot &&
							p.RbWhileTimer.count == 0 &&
							p.RstpVersion &&
							p.Learn &&
							!p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventReRootedAndRbWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						}
					} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
						if !p.Forward &&
							!p.Agreed &&
							!p.Proposing &&
							!p.OperEdge {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if !p.Learning &&
							!p.Forwarding &&
							!p.Synced {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventNotLearningAndNotForwardingAndNotSyncedAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Agreed &&
							!p.Synced {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventAgreedAndNotSyncedAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.OperEdge &&
							!p.Synced {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventOperEdgeAndNotSyncedAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Sync &&
							p.Synced {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventSyncAndSyncedAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.RrWhileTimer.count == 0 &&
							p.ReRoot {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventRrWhileEqualZeroAndReRootAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Sync &&
							!p.Synced &&
							!p.OperEdge &&
							p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventSyncAndNotSyncedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Sync &&
							!p.Synced &&
							!p.OperEdge &&
							p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventSyncAndNotSyncedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.ReRoot &&
							p.RrWhileTimer.count != 0 &&
							!p.OperEdge &&
							p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.ReRoot &&
							p.RrWhileTimer.count != 0 &&
							!p.OperEdge &&
							p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Disputed &&
							!p.OperEdge &&
							p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventDisputedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Disputed &&
							!p.OperEdge &&
							p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventDisputedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.FdWhileTimer.count == 0 &&
							p.RrWhileTimer.count == 0 &&
							!p.Sync &&
							!p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.FdWhileTimer.count == 0 &&
							!p.ReRoot &&
							!p.Sync &&
							!p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndNotLearnSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Agreed &&
							p.RrWhileTimer.count == 0 &&
							!p.Sync &&
							!p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Agreed &&
							!p.ReRoot &&
							!p.Sync &&
							!p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventAgreedAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.OperEdge &&
							p.RrWhileTimer.count == 0 &&
							!p.Sync &&
							!p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.OperEdge &&
							!p.ReRoot &&
							!p.Sync &&
							!p.Learn {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventOperEdgeAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.FdWhileTimer.count == 0 &&
							p.RrWhileTimer.count == 0 &&
							!p.Sync &&
							p.Learn &&
							!p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.FdWhileTimer.count == 0 &&
							!p.ReRoot &&
							!p.Sync &&
							p.Learn &&
							!p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Agreed &&
							p.RrWhileTimer.count == 0 &&
							!p.Sync &&
							p.Learn &&
							!p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Agreed &&
							!p.ReRoot &&
							!p.Sync &&
							p.Learn &&
							!p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventAgreedAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.OperEdge &&
							p.RrWhileTimer.count == 0 &&
							!p.Sync &&
							p.Learn &&
							!p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.OperEdge &&
							!p.ReRoot &&
							!p.Sync &&
							p.Learn &&
							!p.Forward {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventOperEdgeAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
								src: src,
							}
						}
					} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateAlternatePort {
						if p.Proposed &&
							!p.Agree {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.b.AllSynced() &&
							!p.Agree {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Proposed &&
							p.Agree {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.FdWhileTimer.count != int32(p.PortTimes.ForwardingDelay) {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventFdWhileNotEqualForwardDelayAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.Sync {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventSyncAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.ReRoot {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventReRootAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if !p.Synced {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventNotSyncedAndSelectedAndNotUpdtInfo,
								src: src,
							}
						} else if p.RbWhileTimer.count != int32(2*p.PortTimes.HelloTime) &&
							p.Role == PortRoleBackupPort {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventRbWhileNotEqualTwoTimesHelloTimeAndRoleEqualsBackupPortAndSelectedAndNotUpdtInfo,
								src: src,
							}
						}
					} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateBlockPort {
						if !p.Learning &&
							!p.Forwarding {
							p.PrtMachineFsm.PrtEvents <- MachineEvent{
								e:   PrtEventNotLearningAndNotForwardingAndSelectedAndNotUpdtInfo,
								src: src,
							}
						}
					}
				}*/
		}
	}
}

/*
No need to when these parameters change as selected/updtInfo will
be used as the trigger for update
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
*/
func (p *StpPort) NotifySelectedChanged(src string, oldselected bool, newselected bool) {
	// The following machines need to know about
	// changes in Selected State
	// 1) Port Information
	// 2) Port Role Transitions
	// 3) Port Transmit
	if oldselected != newselected {
		StpMachineLogger("INFO", src, p.IfIndex, fmt.Sprintf("NotifySelectedChanged Role[%d] SelectedRole[%d] Forwarding[%t] Learning[%t] Agreed[%t] Agree[%t]\nProposing[%t] OperEdge[%t] Agreed[%t] Agree[%t]\nReRoot[%t] Selected[%t], UpdtInfo[%t] Fdwhile[%d] rrWhile[%d]\n",
			p.Role, p.SelectedRole, p.Forwarding, p.Learning, p.Agreed, p.Agree, p.Proposing, p.OperEdge, p.Synced, p.Sync, p.ReRoot, p.Selected, p.UpdtInfo, p.FdWhileTimer.count, p.RrWhileTimer.count))

		// PI
		if p.Selected {
			if src != PimMachineModuleStr &&
				(p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateAged ||
					p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateCurrent) {
				if p.UpdtInfo {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:   PimEventSelectedAndUpdtInfo,
						src: src,
					}
				}
			}
			/*
				if src != PtxmMachineModuleStr &&
					p.PtxmMachineFsm.Machine.Curr.CurrentState() == PtxmStateIdle &&
					!p.UpdtInfo {
					if p.HelloWhenTimer.count == 0 {
						p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
							e:   PtxmEventHelloWhenEqualsZeroAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.SendRSTP &&
						p.NewInfo &&
						p.TxCount < p.b.TxHoldCount &&
						p.HelloWhenTimer.count != 0 {
						p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
							e:   PtxmEventSendRSTPAndNewInfoAndTxCountLessThanTxHoldCoundAndHelloWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if !p.SendRSTP &&
						p.NewInfo &&
						p.Role == PortRoleRootPort &&
						p.TxCount < p.b.TxHoldCount &&
						p.HelloWhenTimer.count != 0 {
						p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
							e:   PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if !p.SendRSTP &&
						p.NewInfo &&
						p.Role == PortRoleDesignatedPort &&
						p.TxCount < p.b.TxHoldCount &&
						p.HelloWhenTimer.count != 0 {
						p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
							e:   PtxmEventNotSendRSTPAndNewInfoAndDesignatedPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}

				}
			*/
			if src != PrtMachineModuleStr &&
				!p.UpdtInfo {

				if p.SelectedRole == PortRoleDisabledPort &&
					p.Role != p.SelectedRole {
					p.PrtMachineFsm.PrtEvents <- MachineEvent{
						e:   PrtEventSelectedRoleEqualDisabledPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
						src: src,
					}
				}
				if p.SelectedRole == PortRoleRootPort &&
					p.Role != p.SelectedRole {
					p.PrtMachineFsm.PrtEvents <- MachineEvent{
						e:   PrtEventSelectedRoleEqualRootPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
						src: src,
					}
				}
				if p.SelectedRole == PortRoleDesignatedPort &&
					p.Role != p.SelectedRole {
					p.PrtMachineFsm.PrtEvents <- MachineEvent{
						e:   PrtEventSelectedRoleEqualDesignatedPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
						src: src,
					}
				}
				if p.SelectedRole == PortRoleAlternatePort &&
					p.Role != p.SelectedRole {
					p.PrtMachineFsm.PrtEvents <- MachineEvent{
						e:   PrtEventSelectedRoleEqualAlternateAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
						src: src,
					}
				}
				if p.SelectedRole == PortRoleBackupPort &&
					p.Role != p.SelectedRole {
					p.PrtMachineFsm.PrtEvents <- MachineEvent{
						e:   PrtEventSelectedRoleEqualBackupPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
						src: src,
					}
				}
				if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDisablePort &&
					!p.Learning &&
					!p.Forwarding {
					p.PrtMachineFsm.PrtEvents <- MachineEvent{
						e:   PrtEventNotLearningAndNotForwardingAndSelectedAndNotUpdtInfo,
						src: src,
					}
				}
				if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDisabledPort {
					if p.FdWhileTimer.count != int32(p.PortTimes.MaxAge) {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventFdWhileNotEqualMaxAgeAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Sync {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSyncAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.ReRoot {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventReRootAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if !p.Synced {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventNotSyncedAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
				} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateRootPort {
					if p.Proposed &&
						!p.Agree {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.b.AllSynced() &&
						!p.Agree {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Proposed &&
						p.Agree {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if !p.Forward &&
						!p.ReRoot {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventNotForwardAndNotReRootAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.SelectedRole == PortRoleRootPort &&
						p.Role != p.SelectedRole {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSelectedRoleEqualRootPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.RrWhileTimer.count != int32(p.PortTimes.ForwardingDelay) {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventRrWhileNotEqualFwdDelayAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.ReRoot &&
						p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventReRootAndForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.FdWhileTimer.count == 0 &&
						p.RstpVersion &&
						!p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventFdWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.ReRoot &&
						p.RbWhileTimer.count == 0 &&
						p.RstpVersion &&
						!p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventReRootedAndRbWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.FdWhileTimer.count == 0 &&
						p.RstpVersion &&
						p.Learn &&
						!p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventFdWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.ReRoot &&
						p.RbWhileTimer.count == 0 &&
						p.RstpVersion &&
						p.Learn &&
						!p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventReRootedAndRbWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
				} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
					if !p.Forward &&
						!p.Agreed &&
						!p.Proposing &&
						!p.OperEdge {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if !p.Learning &&
						!p.Forwarding &&
						!p.Synced {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventNotLearningAndNotForwardingAndNotSyncedAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Agreed &&
						!p.Synced {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventAgreedAndNotSyncedAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.OperEdge &&
						!p.Synced {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventOperEdgeAndNotSyncedAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Sync &&
						p.Synced {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSyncAndSyncedAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.RrWhileTimer.count == 0 &&
						p.ReRoot {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventRrWhileEqualZeroAndReRootAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Sync &&
						!p.Synced &&
						!p.OperEdge &&
						p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSyncAndNotSyncedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Sync &&
						!p.Synced &&
						!p.OperEdge &&
						p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSyncAndNotSyncedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.ReRoot &&
						p.RrWhileTimer.count != 0 &&
						!p.OperEdge &&
						p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.ReRoot &&
						p.RrWhileTimer.count != 0 &&
						!p.OperEdge &&
						p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Disputed &&
						!p.OperEdge &&
						p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventDisputedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Disputed &&
						!p.OperEdge &&
						p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventDisputedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.FdWhileTimer.count == 0 &&
						p.RrWhileTimer.count == 0 &&
						!p.Sync &&
						!p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.FdWhileTimer.count == 0 &&
						!p.ReRoot &&
						!p.Sync &&
						!p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndNotLearnSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Agreed &&
						p.RrWhileTimer.count == 0 &&
						!p.Sync &&
						!p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Agreed &&
						!p.ReRoot &&
						!p.Sync &&
						!p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventAgreedAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.OperEdge &&
						p.RrWhileTimer.count == 0 &&
						!p.Sync &&
						!p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.OperEdge &&
						!p.ReRoot &&
						!p.Sync &&
						!p.Learn {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventOperEdgeAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.FdWhileTimer.count == 0 &&
						p.RrWhileTimer.count == 0 &&
						!p.Sync &&
						p.Learn &&
						!p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.FdWhileTimer.count == 0 &&
						!p.ReRoot &&
						!p.Sync &&
						p.Learn &&
						!p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Agreed &&
						p.RrWhileTimer.count == 0 &&
						!p.Sync &&
						p.Learn &&
						!p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Agreed &&
						!p.ReRoot &&
						!p.Sync &&
						p.Learn &&
						!p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventAgreedAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.OperEdge &&
						p.RrWhileTimer.count == 0 &&
						!p.Sync &&
						p.Learn &&
						!p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.OperEdge &&
						!p.ReRoot &&
						!p.Sync &&
						p.Learn &&
						!p.Forward {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventOperEdgeAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
				} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateAlternatePort {
					if p.Proposed &&
						!p.Agree {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.b.AllSynced() &&
						!p.Agree {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Proposed &&
						p.Agree {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.FdWhileTimer.count != int32(p.PortTimes.ForwardingDelay) {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventFdWhileNotEqualForwardDelayAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.Sync {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventSyncAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.ReRoot {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventReRootAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if !p.Synced {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventNotSyncedAndSelectedAndNotUpdtInfo,
							src: src,
						}
					} else if p.RbWhileTimer.count != int32(2*p.PortTimes.HelloTime) &&
						p.Role == PortRoleBackupPort {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventRbWhileNotEqualTwoTimesHelloTimeAndRoleEqualsBackupPortAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
				} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateBlockPort {
					if !p.Learning &&
						!p.Forwarding {
						p.PrtMachineFsm.PrtEvents <- MachineEvent{
							e:   PrtEventNotLearningAndNotForwardingAndSelectedAndNotUpdtInfo,
							src: src,
						}
					}
				}
			}
		}
	}
}

func (p *StpPort) NotifyOperEdgeChanged(src string, oldoperedge bool, newoperedge bool) {
	// The following machines need to know about
	// changes in OperEdge State
	// 1) Port Role Transitions
	// 2) Bridge Detection
	if oldoperedge != newoperedge {
		// Prt update 17.29.3
		if p.PrtMachineFsm != nil &&
			p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
			src != PrtMachineModuleStr {
			if !p.Forward &&
				!p.Agreed &&
				!p.Proposing &&
				!p.OperEdge &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.OperEdge &&
				!p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventOperEdgeAndNotSyncedAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.Sync &&
				!p.Synced &&
				!p.OperEdge &&
				p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSyncAndNotSyncedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.Sync &&
				!p.Synced &&
				!p.OperEdge &&
				p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSyncAndNotSyncedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.ReRoot &&
				p.RrWhileTimer.count != 0 &&
				!p.OperEdge &&
				p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.ReRoot &&
				p.RrWhileTimer.count != 0 &&
				!p.OperEdge &&
				p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.Disputed &&
				!p.OperEdge &&
				p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventDisputedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.Disputed &&
				!p.OperEdge &&
				p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventDisputedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.OperEdge &&
				p.RrWhileTimer.count == 0 &&
				!p.Sync &&
				!p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.OperEdge &&
				!p.ReRoot &&
				!p.Sync &&
				!p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventOperEdgeAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.OperEdge &&
				p.RrWhileTimer.count == 0 &&
				!p.Sync &&
				p.Learn &&
				!p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.OperEdge &&
				!p.ReRoot &&
				!p.Sync &&
				p.Learn &&
				!p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventOperEdgeAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
					src: src,
				}
			}
		}
		// Bdm
		if p.BdmMachineFsm.Machine.Curr.CurrentState() == BdmStateEdge &&
			src != BdmMachineModuleStr {
			if !p.OperEdge {
				p.BdmMachineFsm.BdmEvents <- MachineEvent{
					e:   BdmEventNotOperEdge,
					src: src,
				}
			}
		}

	}
}

func (p *StpPort) NotifySelectedRoleChanged(src string, oldselectedrole PortRole, newselectedrole PortRole) {

	if oldselectedrole != newselectedrole {
		StpMachineLogger("INFO", "PRSM", p.IfIndex, fmt.Sprintf("NotifySelectedRoleChange: role[%d] selectedRole[%d]", p.Role, p.SelectedRole))
		if p.Role != p.SelectedRole {
			if p.SelectedRole == PortRoleDisabledPort {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSelectedRoleEqualDisabledPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.SelectedRole == PortRoleRootPort {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSelectedRoleEqualRootPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.SelectedRole == PortRoleDesignatedPort {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSelectedRoleEqualDesignatedPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.SelectedRole == PortRoleAlternatePort {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSelectedRoleEqualAlternateAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
					src: src,
				}
			} else if p.SelectedRole == PortRoleBackupPort {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSelectedRoleEqualBackupPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo,
					src: src,
				}
			}
		}
	}
}

func (p *StpPort) NotifyProposingChanged(src string, oldproposing bool, newproposing bool) {
	if oldproposing != newproposing {
		if src != BdmMachineModuleStr {
			if p.BdmMachineFsm.Machine.Curr.CurrentState() == BdmStateNotEdge {
				if p.EdgeDelayWhileTimer.count == 0 &&
					p.AutoEdgePort &&
					p.SendRSTP &&
					p.Proposing {
					p.BdmMachineFsm.BdmEvents <- MachineEvent{
						e:   BdmEventEdgeDelayWhileEqualZeroAndAutoEdgeAndSendRSTPAndProposing,
						src: PrtMachineModuleStr,
					}
				}
			}
		}
		if src != PrsMachineModuleStr {
			if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
				if !p.Forward &&
					!p.Agreed &&
					!p.Proposing &&
					!p.OperEdge &&
					p.Selected &&
					!p.UpdtInfo {
					p.PrtMachineFsm.PrtEvents <- MachineEvent{
						e:   PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo,
						src: PrtMachineModuleStr,
					}
				}
			}
		}

	}
}

func (p *StpPort) EdgeDelay() uint16 {
	if p.OperPointToPointMAC {
		return MigrateTimeDefault
	} else {
		return p.b.RootTimes.MaxAge
	}
}
