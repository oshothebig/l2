//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//

// config
package lacp

import (
	"fmt"
	//"sync"
	"errors"
	"l2/lacp/protocol/utils"
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

// LaAggConfigParamCheck will validate the config from the user after it has
// been translated to something the Lacp module expects.  Thus if translation
// layer fails it should produce an invalid value.  The error returned
// will be translated to model values
func LaAggConfigParamCheck(ac *LaAggConfig) error {

	if _, ok := utils.PortConfigMap[int32(ac.Id)]; !ok {
		return errors.New(fmt.Sprintln("ERROR Invalid Port Id supplied", ac.Id))
	}

	if ac.Type != LaAggTypeLACP &&
		ac.Type != LaAggTypeSTATIC {
		return errors.New("ERROR Invalid Lag Type Configured Should be LACP(0) or STATIC(1)")
	}
	if ac.Lacp.Interval != LacpSlowPeriodicTime &&
		ac.Lacp.Interval != LacpFastPeriodicTime {
		return errors.New("ERROR Invalid Interval Configured Should be SLOW(1) or FAST(0)")
	}
	if ac.Lacp.Mode != LacpModeActive &&
		ac.Lacp.Mode != LacpModePassive {
		return errors.New("ERROR Invalid LACP Mode Configured Should be ACTIVE(0) or PASSIVE(1)")
	}

	if ac.HashMode != 0 &&
		ac.HashMode != 1 &&
		ac.HashMode != 2 {
		return errors.New("ERROR Invalid LACP Mode Configured Should be LAYER2(0) or LAYER3_4(2) or LAYER2_3(1)")
	}
	return nil
}

// SaveLaAggConfig save off the current configuration data as supplied by the user
func SaveLaAggConfig(ac *LaAggConfig) {
	var a *LaAggregator
	if LaFindAggByName(ac.Name, &a) {
		netMac, _ := net.ParseMAC(ac.Lacp.SystemIdMac)
		sysId := LacpSystem{
			Actor_System:          convertNetHwAddressToSysIdKey(netMac),
			Actor_System_priority: ac.Lacp.SystemPriority,
		}
		a.AggName = ac.Name
		a.AggId = ac.Id
		a.AggMacAddr = sysId.Actor_System
		a.AggPriority = ac.Lacp.SystemPriority
		a.ActorAdminKey = ac.Key
		a.AggType = ac.Type
		a.AggMinLinks = ac.MinLinks
		a.Config = ac.Lacp
		a.LagHash = ac.HashMode
	}
}

func CreateLaAgg(agg *LaAggConfig) {

	//var wg sync.WaitGroup

	a := NewLaAggregator(agg)
	if a != nil {
		a.LacpAggLog(fmt.Sprintf("%#v\n", a))

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
		a.LacpAggLog(fmt.Sprintln("looking for ports with ActorAdminKey", a.ActorAdminKey))
		if mac, err := net.ParseMAC(a.Config.SystemIdMac); err == nil {
			if sgi := LacpSysGlobalInfoByIdGet(LacpSystem{Actor_System: convertNetHwAddressToSysIdKey(mac),
				Actor_System_priority: a.Config.SystemPriority}); sgi != nil {
				for index != -1 {
					if LaFindPortByKey(a.ActorAdminKey, &index, &p) {
						if p.aggSelected == LacpAggUnSelected {
							AddLaAggPortToAgg(a.ActorAdminKey, p.PortNum)

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
		if p != nil {
			p.LaPortLog(fmt.Sprint("Port mode", port.Mode))
			// Is lacp enabled or not
			if port.Mode != LacpModeOn {
				p.lacpEnabled = true
				// make the port aggregatable
				LacpStateSet(&p.ActorAdmin.State, LacpStateAggregationBit)
				// set the activity State
				if port.Mode == LacpModeActive {
					LacpStateSet(&p.ActorAdmin.State, LacpStateActivityBit)
				} else {
					LacpStateClear(&p.ActorAdmin.State, LacpStateActivityBit)
				}
			} else {
				// port is not aggregatible
				LacpStateClear(&p.ActorAdmin.State, LacpStateAggregationBit)
				LacpStateClear(&p.ActorAdmin.State, LacpStateActivityBit)
				p.lacpEnabled = false
			}

			if port.Timeout == LacpShortTimeoutTime {
				LacpStateSet(&p.ActorAdmin.State, LacpStateTimeoutBit)
				// set the oper state to be that of the admin until
				// the fist packet has been received
				LacpStateSet(&p.ActorOper.State, LacpStateTimeoutBit)
			} else {
				LacpStateClear(&p.ActorAdmin.State, LacpStateTimeoutBit)
				// set the oper state to be that of the admin until
				// the fist packet has been received
				LacpStateSet(&p.ActorOper.State, LacpStateTimeoutBit)
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
					AddLaAggPortToAgg(a.ActorAdminKey, p.PortNum)
				}
			}

			if linkStatus && port.Enable {
				// if port is enabled and lacp is enabled
				p.LaAggPortEnabled()

			}
			// check for selection
			p.checkConfigForSelection()

			p.LaPortLog(fmt.Sprintf("PORT Config:\n%#v\n", port))
			p.LaPortLog(fmt.Sprintf("PORT (after config create):\n%#v\n", p))
		}
	} else {
		utils.GlobalLogger.Err("CONF: ERROR PORT ALREADY EXISTS")
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

		DrniEnabled := ((p.DrniName != "" && p.DrniSynced) || p.DrniName == "")
		if DrniEnabled &&
			p.IsPortOperStatusUp() &&
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
		p.LaPortLog(fmt.Sprintln("Set LACP Mode: PrevMode", prevMode, "NewMode", mode))

		// Update the transmission mode
		if mode != prevMode &&
			mode == LacpModeOn {
			p.LaAggPortLacpDisable()

			// lets re-initialize the state machines
			p.BEGIN(true)

			// force check for selection as we should not receive a packet to
			// force the selection processes
			p.checkConfigForSelection()

			// Actor/Partner Aggregation == true
			// agg individual
			// partner admin Key, port == actor admin Key, port
			//if p.MuxMachineFsm.Machine.Curr.CurrentState() == LacpMuxmStateDetached ||
			//	p.MuxMachineFsm.Machine.Curr.CurrentState() == LacpMuxmStateCDetached {
			//	//lets check for selection
			//	p.checkConfigForSelection()
			//}

		} else if mode != prevMode &&
			prevMode == LacpModeOn {
			p.LaAggPortLacpEnabled(mode)
		} else if mode != prevMode {
			if mode == LacpModeActive {
				LacpStateSet(&p.ActorAdmin.State, LacpStateActivityBit)
				// must also set the operational State
				LacpStateSet(&p.ActorOper.State, LacpStateActivityBit)

				// force the next state
				p.PtxMachineFsm.PtxmEvents <- utils.MachineEvent{
					E:   LacpPtxmEventUnconditionalFallthrough,
					Src: PortConfigModuleStr}

			} else {
				LacpStateClear(&p.ActorAdmin.State, LacpStateActivityBit)
				// must also set the operational State
				LacpStateClear(&p.ActorOper.State, LacpStateActivityBit)
				// we are now passive, is the peer passive as well?
				if !LacpStateIsSet(p.PartnerOper.State, LacpStateActivityBit) {
					p.PtxMachineFsm.PtxmEvents <- utils.MachineEvent{
						E:   LacpPtxmEventActorPartnerOperActivityPassiveMode,
						Src: PortConfigModuleStr}
				}
			}
			// state change lets update ntt
			p.TxMachineFsm.TxmEvents <- utils.MachineEvent{
				E:   LacpTxmEventNtt,
				Src: PortConfigModuleStr}
		}
	} else {
		p.LaPortLog(fmt.Sprintln("SetLaAggPortLacpMode: unabled to find port", pId))
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
			LacpStateSet(&p.ActorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateSet(&p.ActorOper.State, LacpStateTimeoutBit)
		} else {
			LacpStateClear(&p.ActorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateClear(&p.ActorOper.State, LacpStateTimeoutBit)
		}
		if timeoutTime, ok := rxm.CurrentWhileTimerValid(); !ok {
			rxm.CurrentWhileTimerTimeoutSet(timeoutTime)
			rxm.CurrentWhileTimerStart()
		}
		// state change lets update ntt
		p.TxMachineFsm.TxmEvents <- utils.MachineEvent{
			E:   LacpTxmEventNtt,
			Src: PortConfigModuleStr}
	}
}

func SetLaAggPortSystemInfo(pId uint16, sysIdMac string, sysPrio uint16) {
	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		mac, ok := net.ParseMAC(sysIdMac)
		if ok == nil {
			macArr := convertNetHwAddressToSysIdKey(mac)
			p.LaAggPortActorAdminInfoSet(macArr, sysPrio)

			if p.IsPortOperStatusUp() &&
				p.aggSelected == LacpAggUnSelected {
				p.checkConfigForSelection()
			}
		}
	}
}

// SetLaAggPortSystemInfoFromDistributedRelay called by DRCP when the Distributed
// relay is linked with the Aggregator, which means that the Agg will now use
// the DR params for the Aggregator port.
//
// TODO this function may need to change to include the operkey change as well as
// change the port Id which is sent on the wire
func SetLaAggPortSystemInfoFromDistributedRelay(pId uint16, sysIdMac string, sysPrio uint16, drName string, synced bool) {
	var p *LaAggPort
	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		mac, ok := net.ParseMAC(sysIdMac)
		if ok == nil {
			p.DrniName = drName
			p.DrniSynced = true

			macArr := convertNetHwAddressToSysIdKey(mac)
			p.LaAggPortActorAdminInfoSet(macArr, sysPrio)
		}
	}
}

// SetLaAggPortCheckSelectionDistributedRelayIsSynced is called by DRCP when the
// Distributed Relay has reached sync state, which should be the trigger to
// allow the local lag to start sycing with the peer device
func SetLaAggPortCheckSelectionDistributedRelayIsSynced(pId uint16, sync bool) {
	mEvtChan := make([]chan utils.MachineEvent, 0)
	evt := make([]utils.MachineEvent, 0)
	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		// indicate that the peer has been synced
		p.DrniSynced = sync
		if p.DrniSynced &&
			p.IsPortOperStatusUp() &&
			p.aggSelected == LacpAggUnSelected {
			p.checkConfigForSelection()
		} else if p.IsPortOperStatusUp() &&
			p.aggSelected == LacpAggSelected {

			p.aggSelected = LacpAggUnSelected
			// partner info should be wrong so lets force sync to be off
			LacpStateClear(&p.PartnerOper.State, LacpStateSyncBit)

			mEvtChan = append(mEvtChan, p.MuxMachineFsm.MuxmEvents)
			evt = append(evt, utils.MachineEvent{
				E:   LacpMuxmEventSelectedEqualUnselected,
				Src: PortConfigModuleStr})

			// unselected event
			p.DistributeMachineEvents(mEvtChan, evt, true)
		}
	}
}

func SetLaAggHashMode(aggId int, hashmode uint32) {
	var a *LaAggregator
	if LaFindAggById(aggId, &a) {
		a.LagHash = hashmode
		if len(a.DistributedPortNumList) > 0 {
			for _, client := range utils.GetAsicDPluginList() {
				err := client.UpdateLag(a.HwAggId, asicDHashModeGet(hashmode), asicDPortBmpFormatGet(a.DistributedPortNumList))
				if err != nil {
					a.LacpAggLog(fmt.Sprintln("SetLaAggHashMode: Error updating LAG in HW", err))
				}
			}
		} else {
			a.LacpAggLog("SetLaAggHashMode: Agg not active in HW")
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

		p.LaPortLog(fmt.Sprintf("Adding LaAggPort %d to LaAgg %d", pId, a.ActorAdminKey))
		// add port to port number list
		a.PortNumList = append(a.PortNumList, p.PortNum)
		// add reference to aggId
		p.AggId = a.AggId

		// attach the port to the aggregator
		//LacpStateSet(&p.ActorAdmin.State, LacpStateAggregationBit)

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
		p.LaPortLog(fmt.Sprintln("deleting port from agg portList", pId, a.PortNumList))

		LacpStateClear(&p.ActorAdmin.State, LacpStateAggregationBit)

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
