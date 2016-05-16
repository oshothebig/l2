// config
package lacp

import (
	"fmt"
	//"sync"
	"net"
	"time"
)

const (
	LaAggTypeLACP = iota + 1
	LaAggTypeSTATIC
)

const PortConfigModuleStr = "Port Config"

// 802.1.AX-2014 7.3.1.1 Aggregator attributes GET-SET
type AggConfig struct {
	// GET-SET
	AggName string
	// GET-SET
	AggActorSystemID [6]uint8
	// GET-SET
	AggActorSystemPriority uint16
	// GET-SET
	AggActorAdminKey uint16
	// GET-SET   up/down enum
	AggAdminState bool
	// GET-SET  enable/disable enum
	AggLinkUpDownNotificationEnable bool
	// GET-SET 10s of microseconds
	AggCollectorMaxDelay uint16
	// GET-SET
	AggPortAlgorithm [3]uint8
	// GET-SET
	AggPartnerAdminPortAlgorithm [3]uint8
	// GET-SET up to 4096 values conversationids
	AggConversationAdminLink []int
	// GET-SET
	AggPartnerAdminPortConverstaionListDigest [16]uint8
	// GET-SET
	AggAdminDiscardWrongConversation bool
	// GET-SET 4096 values
	AggAdminServiceConversationMap []int
	// GET-SET
	AggPartnerAdminConvServiceMappingDigest [16]uint8
}

type LacpConfigInfo struct {
	Interval time.Duration
	Mode     uint32
	// In format AA:BB:CC:DD:EE:FF
	SystemIdMac    string
	SystemPriority uint16
}

type LaAggConfig struct {
	// Aggregator name
	Name string
	// Aggregator_MAC_address
	Mac [6]uint8
	// Aggregator_Identifier
	Id int
	// Actor_Admin_Aggregator_Key
	Key uint16
	// Aggregator Type, LACP or STATIC
	Type uint32
	// Minimum number of links
	MinLinks uint16
	// Enabled
	Enabled bool
	// LAG_ports
	LagMembers []uint16

	// System to attach this agg to
	Lacp LacpConfigInfo

	// mau properties of each link
	Properties PortProperties

	// hash config
	HashMode uint32
}

type AggPortConfig struct {
	// GET-SET
	AggPortActorSystemPriority uint16
	// GET-SET
	AggPortActorAdminKey uint16
	// GET-SET
	AggPortPartnerAdminSystemPriority uint16
	// GET-SET
	AggPortPartnerAdminSystemId [6]uint8
	// GET-SET
	AggPortPartnerAdminKey uint16
	// GET-SET
	AggPortActorPortPriority uint8
	// GET-SET
	AggPortPartnerAdminPort int
	// GET-SET
	AggPortPartnerAdminPortPriority uint8
	// GET-SET
	AggPortActorAdminState uint8
	// GET-SET
	AggPortPartnerAdminState uint8
	// GET-SET
	AggPortLinkNumberID int
	// GET-SET
	AggPortPartnerAdminLInkNumberID int
	// GET-SET
	AggPortWTRTime int
	// GET-SET
	AggPortProtocolDA [6]uint8
}

type LaAggPortConfig struct {

	// Actor_Port_Number
	Id uint16
	// Actor_Port_priority
	Prio uint16
	// Actor Admin Key
	Key uint16
	// Actor Oper Key
	//OperKey uint16
	// Actor_Port_Aggregator_Identifier
	AggId int

	// Admin Enable/Disable
	Enable bool

	// lacp mode On/Active/Passive
	Mode int

	// lacp timeout SHORT/LONG
	Timeout time.Duration

	// Port capabilities and attributes
	Properties PortProperties

	// Linux If
	TraceEna bool
	IntfId   string
}

func SaveLaAggConfig(ac *LaAggConfig) {
	var a *LaAggregator
	if LaFindAggByName(ac.Name, &a) {
		netMac, _ := net.ParseMAC(ac.Lacp.SystemIdMac)
		sysId := LacpSystem{
			actor_System:          convertNetHwAddressToSysIdKey(netMac),
			Actor_System_priority: ac.Lacp.SystemPriority,
		}
		a.AggName = ac.Name
		a.AggId = ac.Id
		a.aggMacAddr = sysId.actor_System
		a.actorAdminKey = ac.Key
		a.AggType = ac.Type
		a.AggMinLinks = ac.MinLinks
		a.Config = ac.Lacp
		a.LagHash = ac.HashMode
	}
}

func CreateLaAgg(agg *LaAggConfig) {

	//var wg sync.WaitGroup

	a := NewLaAggregator(agg)
	a.LacpDebug.logger.Info(fmt.Sprintf("%#v\n", a))

	/*
		// two methods for creating ports after CreateLaAgg is created
		// 1) PortNumList is populated
		// 2) find Key's that match
		for _, pId := range a.PortNumList {
			wg.Add(1)
			go func(pId uint16) {
				var p *LaAggPort
				defer wg.Done()
				if LaFindPortById(pId, &p) && p.aggSelected == LacpAggUnSelected {
					// if aggregation has been provided then lets kick off the process
					p.checkConfigForSelection()
				}
			}(pId)
		}

		wg.Wait()
	*/
	index := 0
	var p *LaAggPort
	a.LacpDebug.logger.Info(fmt.Sprintf("looking for ports with actorAdminKey\n", a.actorAdminKey))
	if mac, err := net.ParseMAC(a.Config.SystemIdMac); err == nil {
		if sgi := LacpSysGlobalInfoByIdGet(LacpSystem{actor_System: convertNetHwAddressToSysIdKey(mac),
			Actor_System_priority: a.Config.SystemPriority}); sgi != nil {
			for index != -1 {
				if LaFindPortByKey(a.actorAdminKey, &index, &p) {
					if p.aggSelected == LacpAggUnSelected {
						AddLaAggPortToAgg(a.actorAdminKey, p.PortNum)

						if p.PortEnabled {
							p.checkConfigForSelection()
						}

					}
				} else {
					break
				}
			}
		}
	}
}

func DeleteLaAgg(Id int) {
	var a *LaAggregator
	if LaFindAggById(Id, &a) {

		for _, pId := range a.PortNumList {
			DeleteLaAggPort(pId)
		}
		a.DeleteLaAgg()
	}
}

func EnableLaAgg(Id int) {
	var a *LaAggregator
	if LaFindAggById(Id, &a) {

		for _, pId := range a.PortNumList {
			EnableLaAggPort(pId)
		}
	}
}

func DisableLaAgg(Id int) {
	var a *LaAggregator
	if LaFindAggById(Id, &a) {

		for _, pId := range a.PortNumList {
			DisableLaAggPort(pId)
		}
	}
}

func CreateLaAggPort(port *LaAggPortConfig) {
	var pTmp *LaAggPort

	// sanity check that port does not exist already
	if !LaFindPortById(port.Id, &pTmp) {
		p := NewLaAggPort(port)

		p.LacpDebug.logger.Info(fmt.Sprintf("Port mode", port.Mode))
		// Is lacp enabled or not
		if port.Mode != LacpModeOn {
			p.lacpEnabled = true
			// make the port aggregatable
			LacpStateSet(&p.actorAdmin.State, LacpStateAggregationBit)
			// set the activity State
			if port.Mode == LacpModeActive {
				LacpStateSet(&p.actorAdmin.State, LacpStateActivityBit)
			} else {
				LacpStateClear(&p.actorAdmin.State, LacpStateActivityBit)
			}
		} else {
			// port is not aggregatible
			LacpStateClear(&p.actorAdmin.State, LacpStateAggregationBit)
			LacpStateClear(&p.actorAdmin.State, LacpStateActivityBit)
			p.lacpEnabled = false
		}

		if port.Timeout == LacpShortTimeoutTime {
			LacpStateSet(&p.actorAdmin.State, LacpStateTimeoutBit)
		} else {
			LacpStateClear(&p.actorAdmin.State, LacpStateTimeoutBit)
		}

		// lets start all the State machines
		p.BEGIN(false)
		linkStatus := p.IsPortOperStatusUp()
		p.LaPortLog(fmt.Sprintf("Creating LaAggPort %d is link up %t admin up %t", port.Id, linkStatus, port.Enable))

		if p.Key != 0 {
			var a *LaAggregator
			if LaFindAggByKey(p.Key, &a) {
				p.LaPortLog("Found Agg by Key, attaching port to agg")
				// If the agg is defined lets add port to
				AddLaAggPortToAgg(a.actorAdminKey, p.PortNum)
			}
		}

		if linkStatus && port.Enable {
			// if port is enabled and lacp is enabled
			p.LaAggPortEnabled()

		}
		// check for selection
		p.checkConfigForSelection()

		p.LacpDebug.logger.Info(fmt.Sprintf("PORT Config:\n%#v\n", port))
		p.LacpDebug.logger.Info(fmt.Sprintf("PORT (after config create):\n%#v\n", p))
	} else {
		fmt.Println("CONF: ERROR PORT ALREADY EXISTS")
	}
}

func DeleteLaAggPort(pId uint16) {
	var p *LaAggPort
	if LaFindPortById(pId, &p) {
		// detech the port from sw
		DeleteLaAggPortFromAgg(p.Key, pId)
		// finally delete the stop all machines
		// and delete the port
		p.LaAggPortDelete()
	} else {
		fmt.Println("CONF: DeleteLaAggPort unable to find port", pId)
	}
}

func DisableLaAggPort(pId uint16) {
	var p *LaAggPort

	// port exists
	// port exists in agg exists
	if LaFindPortById(pId, &p) {
		p.LaAggPortDisable()
	} else {
		fmt.Println("ERROR DisableLaAggPort, did not find port", pId)
	}
}

func EnableLaAggPort(pId uint16) {
	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) &&
		//p.aggSelected == LacpAggUnSelected &&
		LaAggPortNumListPortIdExist(p.Key, pId) {
		p.LaAggPortEnabled()

		if p.IsPortOperStatusUp() &&
			p.aggSelected == LacpAggUnSelected {
			p.checkConfigForSelection()
		}
	}
}

// SetLaAggPortLacpMode will set the various
// lacp modes - On, Active, Passive
func SetLaAggPortLacpMode(pId uint16, mode int) {

	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		prevMode := LacpModeGet(p.ActorOper.State, p.lacpEnabled)
		p.LaPortLog(fmt.Sprintf("PrevMode", prevMode, "NewMode", mode))

		// Update the transmission mode
		if mode != prevMode &&
			mode == LacpModeOn {
			p.LaAggPortLacpDisable()

			// Actor/Partner Aggregation == true
			// agg individual
			// partner admin Key, port == actor admin Key, port
			if p.MuxMachineFsm.Machine.Curr.CurrentState() == LacpMuxmStateDetached ||
				p.MuxMachineFsm.Machine.Curr.CurrentState() == LacpMuxmStateCDetached {
				// lets check for selection
				p.checkConfigForSelection()
			}

		} else if mode != prevMode &&
			prevMode == LacpModeOn {
			p.LaAggPortLacpEnabled(mode)
		} else if mode != prevMode {
			if mode == LacpModeActive {
				LacpStateSet(&p.actorAdmin.State, LacpStateActivityBit)
				// must also set the operational State
				LacpStateSet(&p.ActorOper.State, LacpStateActivityBit)

				// force the next state
				p.PtxMachineFsm.PtxmEvents <- LacpMachineEvent{e: LacpPtxmEventUnconditionalFallthrough,
					src: PortConfigModuleStr}

			} else {
				LacpStateClear(&p.actorAdmin.State, LacpStateActivityBit)
				// must also set the operational State
				LacpStateClear(&p.ActorOper.State, LacpStateActivityBit)
				// we are now passive, is the peer passive as well?
				if !LacpStateIsSet(p.PartnerOper.State, LacpStateActivityBit) {
					p.PtxMachineFsm.PtxmEvents <- LacpMachineEvent{e: LacpPtxmEventActorPartnerOperActivityPassiveMode,
						src: PortConfigModuleStr}
				}
			}
			// state change lets update ntt
			p.TxMachineFsm.TxmEvents <- LacpMachineEvent{e: LacpTxmEventNtt,
				src: PortConfigModuleStr}
		}
	}
}

// SetLaAggPortLacpPeriod will set the periodic rate at which a packet should
// be transmitted.  What this actually means is at what rate the peer should
// transmit a packet to us.
// FAST and SHORT are the periods, the lacp state timeout is encoded such
// that FAST  is 1 and SHORT is 0
func SetLaAggPortLacpPeriod(pId uint16, period time.Duration) {

	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		rxm := p.RxMachineFsm
		p.LaPortLog(fmt.Sprintf("NewPeriod", period))

		// lets set the period
		if period == LacpFastPeriodicTime {
			LacpStateSet(&p.actorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateSet(&p.ActorOper.State, LacpStateTimeoutBit)
		} else {
			LacpStateClear(&p.actorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateClear(&p.ActorOper.State, LacpStateTimeoutBit)
		}
		if timeoutTime, ok := rxm.CurrentWhileTimerValid(); !ok {
			rxm.CurrentWhileTimerTimeoutSet(timeoutTime)
			rxm.CurrentWhileTimerStart()
		}
		// state change lets update ntt
		p.TxMachineFsm.TxmEvents <- LacpMachineEvent{e: LacpTxmEventNtt,
			src: PortConfigModuleStr}
	}
}

// SetLaAggPortLacpTimeout will set the timeout at which a packet should
// declare a link down.
// SHORT and LONG are the periods, the lacp state timeout is encoded such
// that SHORT is 3 * FAST PERIODIC TIME and LONG is 3 * SLOW PERIODIC TIME
func SetLaAggPortLacpTimeout(pId uint16, timeout time.Duration) {

	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		rxm := p.RxMachineFsm
		// lets set the period
		if timeout == LacpShortTimeoutTime {
			LacpStateSet(&p.actorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateSet(&p.ActorOper.State, LacpStateTimeoutBit)
		} else {
			LacpStateClear(&p.actorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateClear(&p.ActorOper.State, LacpStateTimeoutBit)
		}
		if timeoutTime, ok := rxm.CurrentWhileTimerValid(); !ok {
			rxm.CurrentWhileTimerTimeoutSet(timeoutTime)
		}
	}
}

func SetLaAggPortSystemInfo(pId uint16, sysIdMac string, sysPrio uint16) {
	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		mac, _ := net.ParseMAC(sysIdMac)
		macArr := convertNetHwAddressToSysIdKey(mac)
		p.LaAggPortActorAdminInfoSet(macArr, sysPrio)

		if p.IsPortOperStatusUp() &&
			p.aggSelected == LacpAggUnSelected {
			p.checkConfigForSelection()
		}
	}
}

func SetLaAggHashMode(aggId int, hashmode uint32) {
	var a *LaAggregator
	if LaFindAggById(aggId, &a) {
		a.LagHash = hashmode
		if len(a.DistributedPortNumList) > 0 {
			asicDUpdateLag(a)
		} else {
			a.LacpDebug.logger.Info("SetLaAggHashMode: Agg not active in HW")
		}
	} else {
		fmt.Println("SetLaAggHashMode: Unable to find aggId", aggId)
	}
}

func AddLaAggPortToAgg(Key uint16, pId uint16) {

	var a *LaAggregator
	var p *LaAggPort

	// both add and port must have existed
	if LaFindAggByKey(Key, &a) && LaFindPortById(pId, &p) &&
		p.aggSelected == LacpAggUnSelected &&
		!LaAggPortNumListPortIdExist(Key, pId) {

		p.LaPortLog(fmt.Sprintf("Adding LaAggPort %d to LaAgg %d", pId, a.actorAdminKey))
		// add port to port number list
		a.PortNumList = append(a.PortNumList, p.PortNum)
		// add reference to aggId
		p.AggId = a.AggId

		// attach the port to the aggregator
		//LacpStateSet(&p.actorAdmin.State, LacpStateAggregationBit)

		// Port is now aggregatible
		//LacpStateSet(&p.ActorOper.State, LacpStateAggregationBit)

		// well obviously this should pass
		//p.checkConfigForSelection()
	}
}

func DeleteLaAggPortFromAgg(Key uint16, pId uint16) {

	var a *LaAggregator
	var p *LaAggPort

	// both add and port must have existed
	if LaFindAggByKey(Key, &a) && LaFindPortById(pId, &p) &&
		//p.aggSelected == LacpAggSelected &&
		LaAggPortNumListPortIdExist(Key, pId) {
		p.LaPortLog(fmt.Sprintf("deleting port from agg portList", pId, a.PortNumList))

		LacpStateClear(&p.actorAdmin.State, LacpStateAggregationBit)

		// disable the port
		p.LaAggPortDisable()

		// update selection to be unselected
		p.checkConfigForSelection()

		// del reference to aggId
		p.AggId = 0

		// detach the port from the agg port list
		for idx, PortNum := range a.PortNumList {
			if PortNum == pId {
				a.PortNumList = append(a.PortNumList[:idx], a.PortNumList[idx+1:]...)
			}
		}

	}
}

func GetLaAggPortActorOperState(pId uint16) uint8 {
	var p *LaAggPort
	if LaFindPortById(pId, &p) {
		return p.ActorOper.State
	}
	return 0
}

func GetLaAggPortPartnerOperState(pId uint16) uint8 {
	var p *LaAggPort
	if LaFindPortById(pId, &p) {
		return p.PartnerOper.State
	}
	return 0
}
