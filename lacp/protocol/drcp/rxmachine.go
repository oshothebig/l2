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
// 802.1ax-2014 Section 9.4.14 DRCPDU Receive machine
// rxmachine.go
package drcp

import (
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/utils"
	"sort"
	"time"
)

const RxMachineModuleStr = "DRCP Rx Machine"

// drxm States
const (
	RxmStateNone = iota + 1
	RxmStateInitialize
	RxmStateExpired
	RxmStatePortalCheck
	RxmStateCompatibilityCheck
	RxmStateDefaulted
	RxmStateDiscard
	RxmStateCurrent
)

var RxmStateStrMap map[fsm.State]string

func RxMachineStrStateMapCreate() {
	RxmStateStrMap = make(map[fsm.State]string)
	RxmStateStrMap[RxmStateNone] = "None"
	RxmStateStrMap[RxmStateInitialize] = "Initialize"
	RxmStateStrMap[RxmStateExpired] = "Expired"
	RxmStateStrMap[RxmStatePortalCheck] = "Portal Check"
	RxmStateStrMap[RxmStateCompatibilityCheck] = "Compatibility Check"
	RxmStateStrMap[RxmStateDefaulted] = "Defaulted"
	RxmStateStrMap[RxmStateDiscard] = "Discard"
	RxmStateStrMap[RxmStateCurrent] = "Current"
}

// rxm events
const (
	RxmEventBegin = iota + 1
	RxmEventNotIPPPortEnabled
	RxmEventNotDRCPEnabled
	RxmEventIPPPortEnabledAndDRCPEnabled
	RxmEventDRCPDURx
	RxmEventDRCPCurrentWhileTimerExpired
	RxmEventNotDifferPortal
	RxmEventDifferPortal
	RxmEventNotDifferConfPortal
	RxmEventDifferConfPortal
)

type RxDrcpPdu struct {
	pdu          *layers.DRCP
	src          string
	responseChan chan string
}

// DrcpRxMachine holds FSM and current State
// and event channels for State transitions
type RxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *DRCPIpp

	// debug log
	//log chan string

	// timer interval
	currentWhileTimerTimeout time.Duration

	// timers
	currentWhileTimer *time.Timer

	// machine specific events
	RxmEvents     chan utils.MachineEvent
	RxmPktRxEvent chan RxDrcpPdu
}

func (rxm *RxMachine) PrevState() fsm.State { return rxm.PreviousState }

// PrevStateSet will set the previous State
func (rxm *RxMachine) PrevStateSet(s fsm.State) { rxm.PreviousState = s }

// Stop should clean up all resources
func (rxm *RxMachine) Stop() {
	rxm.CurrentWhileTimerStop()

	close(rxm.RxmEvents)
	close(rxm.RxmPktRxEvent)
}

// NewDrcpRxMachine will create a new instance of the LacpRxMachine
func NewDrcpRxMachine(port *DRCPIpp) *RxMachine {
	rxm := &RxMachine{
		p:             port,
		PreviousState: RxmStateNone,
		RxmEvents:     make(chan MachineEvent, 10),
		RxmPktRxEvent: make(chan RxDrcpPdu, 100)}

	port.RxMachineFsm = rxm

	// create then stop
	rxm.CurrentWhileTimerStart()
	rxm.CurrentWhileTimerStop()

	return rxm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (rxm *RxMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if rxm.Machine == nil {
		rxm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	rxm.Machine.Rules = r
	rxm.Machine.Curr = &utils.StateEvent{
		StrStateMap: RxmStateStrMap,
		LogEna:      false,
		Logger:      rxm.DrcpRxmLog,
		Owner:       RxMachineModuleStr,
	}

	return rxm.Machine
}

// DrcpRxMachineInitialize function to be called after
// State transition to INITIALIZE
func (rxm *RxMachine) DrcpRxMachineInitialize(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	// Record default params
	rxm.recordDefaultDRCPDU()

	p.DRFNeighborOperDRCPState.ClearState(layers.DRCPStateIPPActivity)
	// next State
	return RxmStateInitialize
}

// DrcpRxMachineExpired function to be called after
// State transition to EXPIRED
func (rxm *RxMachine) DrcpRxMachineExpired(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	p.DRFNeighborOperDRCPState.ClearState(layers.DRCPStateIPPActivity)
	defer rxm.NotifyDRCPStateTimeoutChange(p.DRFNeighborOperDRCPState.GetState(DRCPStateDRCPTimeout), layers.DRCPShortTimeout)
	// short timeout
	p.DRFNeighborOperDRCPState.SetState(layers.DRCPStateDRCPTimeout)
	// start the timer
	p.CurrentWhileTimerTimeoutSet(DrniShortTimeoutTime)
	p.CurrentWhiletimertimeoutStart()
	return RxmStateExpired
}

// DrcpRxMachinePortalCheck function to be called after
// State transition to PORTAL CHECK
func (rxm *RxMachine) DrcpRxMachinePortalCheck(m fsm.Machine, data interface{}) fsm.State {

	rxm.recordPortalValues()
	return RxmStatePortalCheck
}

// DrcpRxMachineCompatibilityCheck function to be called after
// State transition to COMPATIBILITY CHECK
func (rxm *RxMachine) DrcpRxMachineCompatibilityCheck(m fsm.Machine, data interface{}) fsm.State {

	rxm.recordPortalConfValues()
	return RxmStateCompatibilityCheck
}

// DrcpRxMachineDiscard function to be called after
// State transition to DISCARD
func (rxm *RxMachine) DrcpRxMachineDiscard(m fsm.Machine, data interface{}) fsm.State {

	// NO ACTIONS ACCORDING TO STANDARD
	return RxmStateDiscard
}

// DrcpRxMachineDefaulted function to be called after
// State transition to DEFAULTED
func (rxm *RxMachine) DrcpRxMachineDefaulted(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	p.DRFHomeOperDRCPState.SetState(layers.DRCPStateExpired)
	rxm.recordDefaultDRCPDU(data)
	p.reportToManagement()

	return RxmStateDefaulted
}

// DrcpRxMachineCurrent function to be called after
// State transition to CURRENT
func (rxm *RxMachine) DrcpRxMachineCurrent(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	rxm.recordNeighborState()
	rxm.updateNTT()

	// 1 short , 0 long
	if p.DRFHomeOperDRCPState.GetState(layers.DRCPStateDRCPTimeout) {
		rxm.CurrentWhileTimerTimeoutSet(DrniShortTimeoutTime)
	} else {
		rxm.CurrentWhileTimerTimeoutSet(DrniLongTimeoutTime)
	}

	rxm.CurrentWhileTimerStart()
	p.DRFHomeOperDRCPState.ClearState(DRCPStateExpired)

	return RxmStateCurrent
}

func DrcpRxMachineFSMBuild(p *DRCPIpp) *LacpRxMachine {

	RxMachineStrStateMapCreate()

	rules := fsm.Ruleset{}

	// Instantiate a new LacpRxMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the initalize State
	rxm := NewDrcpRxMachine(p)

	//BEGIN -> INITITIALIZE
	rules.AddRule(RxmStateNone, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateInitialize, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateExpired, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStatePortalCheck, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDefaulted, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDiscard, RxmEventBegin, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCurrent, RxmEventBegin, rxm.DrcpRxMachineInitialize)

	// NOT IPP PORT ENABLED  > INITIALIZE
	rules.AddRule(RxmStateNone, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateInitialize, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateExpired, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStatePortalCheck, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDefaulted, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDiscard, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCurrent, RxmEventNotIPPPortEnabled, rxm.DrcpRxMachineInitialize)

	// NOT DRCP ENABLED  > INITIALIZE
	rules.AddRule(RxmStateNone, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateInitialize, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateExpired, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStatePortalCheck, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDefaulted, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateDiscard, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)
	rules.AddRule(RxmStateCurrent, RxmEventNotDRCPEnabled, rxm.DrcpRxMachineInitialize)

	// IPP PORT ENABLED AND DRCP ENABLED -> EXPIRED
	rules.AddRule(RxmStateInitialize, RxmEventIPPPortEnabledAndDRCPEnabled, rxm.DrcpRxMachineExpired)

	// DRCPDU RX -> PORTAL CHECK
	rules.AddRule(RxmStateExpired, RxmEventDRCPDURx, rxm.DrcpRxMachinePortalCheck)
	rules.AddRule(RxmStateDiscard, RxmEventDRCPDURx, rxm.DrcpRxMachinePortalCheck)
	rules.AddRule(RxmStateCurrent, RxmEventDRCPDURx, rxm.DrcpRxMachinePortalCheck)
	rules.AddRule(RxmStateDefaulted, RxmEventDRCPDURx, rxm.DrcpRxMachinePortalCheck)

	// NOT DIFFER PORTAL -> DISCARD
	rules.AddRule(RxmStatePortalCheck, RxmEventDifferPortal, rxm.DrcpRxMachineDiscard)

	// NOT DIFFER CONF PORTAL -> CURRENT
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventNotDifferConfPortal, rxm.DrcpRxMachineCurrent)

	// DIFFER CONF PORTAL -> DISCARD
	rules.AddRule(RxmStateCompatibilityCheck, RxmEventDifferConfPortal, rxm.DrcpRxMachineDiscard)

	// DRCP CURRENT WHILE TIMER EXPIRED -> DEFAULTED
	rules.AddRule(RxmStateExpired, RxmEventDRCPCurrentWhileTimerExpired, rxm.DrcpRxMachineDefaulted)
	rules.AddRule(RxmStateDiscard, RxmEventDRCPCurrentWhileTimerExpired, rxm.DrcpRxMachineDefaulted)
	rules.AddRule(RxmStateCurrent, RxmEventDRCPCurrentWhileTimerExpired, rxm.DrcpRxMachineExpired)

	// Create a new FSM and apply the rules
	rxm.Apply(&rules)

	return rxm
}

// DrcpRxMachineMain:  802.1ax-2014 Figure 9-23
// Creation of Rx State Machine State transitions and callbacks
// and create go routine to pend on events
func (p *DRCPIpp) DrcpRxMachineMain() {

	// Build the State machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.12 Receive Machine
	rxm := DrcpRxMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	rxm.Machine.Start(rxm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpRxMachine) {
		m.LacpRxmLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.currentWhileTimer.C:
				// special case if we have pending packets in the queue
				// by the time this expires we want to ensure the packet
				// gets processed first as this will clear/restart the timer
				if len(m.RxmPktRxEvent) == 0 {
					m.DrcpRxmLog("Current While Timer Expired")
					m.Machine.ProcessEvent(RxMachineModuleStr, RxmEventDRCPCurrentWhileTimerExpired, nil)
				}

			case event, ok := <-m.RxmEvents:
				if ok {
					rv := m.Machine.ProcessEvent(event.src, event.e, nil)
					if rv == nil {
						/* continue State transition */
						m.Machine.processPostStates()
					}

					if rv != nil {
						m.DrcpRxmLog(strings.Join([]string{error.Error(rv), event.src, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
					}
				}

				// respond to caller if necessary so that we don't have a deadlock
				if event.ResponseChan != nil {
					utils.SendResponse(RxMachineModuleStr, event.ResponseChan)
				}
				if !ok {
					m.LacpRxmLog("Machine End")
					return
				}
			case rx := <-m.RxmPktRxEvent:
				m.Machine.ProcessEvent(RxMachineModuleStr, RxmEventDRCPDURx, rx.pdu)

				// respond to caller if necessary so that we don't have a deadlock
				if rx.responseChan != nil {
					utils.SendResponse(RxMachineModuleStr, rx.responseChan)
				}
			}
		}
	}(rxm)
}

// NotifyDRCPStateTimeoutChange notify the Periodic Transmit Machine of a neighbor state
// timeout change
func (rxm *RxMachine) NotifyDRCPStateTimeoutChange(oldval, newval int32) {

	if oldval != newval {
		if newval == layers.DRCPShortTimeout {
			p.PtxMachineFsm.PtxmEvents <- utils.MachineEvent{
				E:   PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualShortTimeout,
				Src: RxMachineModuleStr,
			}
		} else {
			p.PtxMachineFsm.PtxmEvents <- utils.MachineEvent{
				E:   PtxmEventDRFNeighborOPerDRCPStateTimeoutEqualLongTimeout,
				Src: RxMachineModuleStr,
			}
		}
	}
}

// processPostStates will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStates() {
	rxm.processPostStateInitialize()
	rxm.processPostStateExpire()
	rxm.processPostStatePortalCheck()
	rxm.processPostStateCompatibilityCheck()
	rxm.processPostStateDiscard()
	rxm.processPostStateCurrent()
	rxm.processPostStateDefaulted()
}

// processPostStateInitialize will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateInitialize() {
	p := rxm.p
	if rxm.Machine.Curr.CurrentState() == RxmStateInitialize &&
		p.IPPPortEnabed && p.DRCPEnabled {
		rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventIPPPortEnabledAndDRCPEnabled, nil)
		if rv != nil {
			m.LacpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
		} else {
			rxm.processPostStates()
		}
	}
}

// processPostStateExpired will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateExpired() {
	// nothin to do events are triggered by rx packet or current while timer expired
}

// processPostStatePortalCheck will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStatePortalCheck() {
	p := rxm.p
	if rxm.Machine.Curr.CurrentState() == RxmStatePortalCheck {
		if p.DifferPortal {
			rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventDifferPortal, nil)
			if rv != nil {
				m.LacpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
			} else {
				rxm.processPostStates()
			}
		} else {
			rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventNotDifferPortal, nil)
			if rv != nil {
				m.LacpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
			} else {
				rxm.processPostStates()
			}
		}
	}
}

// processPostStateCompatibilityCheck will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateCompatibilityCheck() {
	p := rxm.p
	if rxm.Machine.Curr.CurrentState() == RxmStateCompatibilityCheck {
		if p.DifferConfPortal {
			rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventDifferConfPortal, nil)
			if rv != nil {
				m.LacpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
			} else {
				rxm.processPostStates()
			}
		} else {
			rv := rxm.Machine.ProcessEvent(RxMachineModuleStr, RxmEventNotDifferConfPortal, nil)
			if rv != nil {
				m.LacpRxmLog(strings.Join([]string{error.Error(rv), RxMachineModuleStr, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
			} else {
				rxm.processPostStates()
			}
		}
	}
}

// processPostStateDiscard will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateDiscard(pdu interface{}) {
	// nothin to do events are triggered by rx packet or current while timer expired
}

// processPostStateCurrent will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateCurrent(pdu interface{}) {
	// nothin to do events are triggered by rx packet or current while timer expired
}

// processPostStateCurrent will check local params to see if any conditions
// are met in order to transition to next state
func (rxm *RxMachine) processPostStateDefaulted(pdu interface{}) {
	// nothin to do events are triggered by rx packet or current while timer expired
}

// updateNTT This function sets NTTDRCPDU to TRUE, if any of:
func (rxm *RxMachine) updateNTT() {
	p := rxm.p
	if !p.DRFHomeOperDRCPState.GetState(layers.DRCPStateGatewaySync) ||
		!p.DRFHomeOperDRCPState.GetState(layers.DRCPStatePortSync) ||
		!p.DRFNeighborOperDRCPState.GetState(layers.DRCPStateGatewaySync) ||
		!p.DRFNeighborOperDRCPState.GetState(layers.DRCPStatePortSync) {
		p.NTTDRCPDU = true
	}
}

// recordDefaultDRCPDU: 802.1ax Section 9.4.1.1
//
// This function sets the current Neighbor Portal System's operational
// parameter values to the default parameter values provided by the
// administrator as follows
func (rxm *RxMachine) recordDefaultDRCPDU(drcpPduInfo *layers.DRCP) {
	p := rxm.p

	p.DRFNeighborPortAlgorithm = p.dr.NeighborAdminPortAlgorithm
	p.DRFNeighborGatewayAlgorithm = p.dr.NeighborAdminGatewayAlgorithm
	p.DRFNeighborConversationPortList = p.dr.DRFNeighborAdminConversationPortListDigest
	p.DRFNeighborConversationGatewayListDigest = p.dr.DRFNeighborAdminConversationGatewayListDigest
	p.DRFNeighborOperDRCPState = p.dr.DRFNeighborAdminDRCPState
	p.DRFNeighborAggregatorPriority = p.dr.a.partnerSystemPriority
	p.DRFNeighborAggregatorId = p.dr.a.partnerSystemId
	p.DrniNeighborPortalPriority = p.dr.a.partnerSystemPriority
	p.DrniNeighborPortalAddr = p.dr.a.partnerSystemId
	p.DRFNeighborPortalSystemNumber = p.DRFHomeConfNeighborPortalSystemNumber
	p.DRFNeighborConfPortalSystemNumber = p.dr.DRFPortalSystemNumber
	p.DRFNeighborState.OpState = false
	p.DRFNeighborState.GatewayVector = false
	p.DRFNeighborState.PortIdList = nil
	p.DRFOtherNeighborState.OpState = false
	p.DRFOtherNeighborState.GatewayVector = false
	p.DRFOtherNeighborState.PortIdList = nil
	p.DRFNeighborAdminAggregatorKey = 0
	p.DRFOtherNeighborAdminAggregatorKey = 0
	p.DRFNeighborOperPartnerAggregatorKey = 0
	p.DRFOtherNeighborOperPartnerAggregatorKey = 0
	if p.DrniThreeSystemPortal {
		for i := 0; i < 1024; i++ {
			p.DrniNeighborGatewayConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber
			p.DrniNeighborPortConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber
		}
	} else {
		for i := 0; i < 1024; i++ {
			p.DrniNeighborGatewayConversation[i] = 1
			if p.ChangePortal {
				p.DrniNeighborPortConversation[i] = 1
			}
		}
	}
	p.CCTimeShared = false
	p.CCEncapTag = false
}

// recordPortalValues: 802.1ax-2014 Section 9.4.11 Functions
//
// This function records the Neighbor’s Portal parameter values carried
// in a received DRCPDU (9.4.3.2) from an IPP, as the current operational
// parameter values for the immediate Neighbor Portal System on this IPP,
// as follows
func (rxm *RxMachine) recordPortalValues(drcpPduInfo *layers.DRCP) {
	p := rxm.p

	p.DRFNeighborAggregatorPriority = drcpPduInfo.PortalInfo.AggPriority
	p.DRFNeighborAggregatorId = drcpPduInfo.PortalInfo.AggId
	p.DrniNeighborPortalPriority = drcpPduInfo.PortalInfo.PortalPriority
	p.DrniNeighborPortalAddr = drcpPduInfo.PortalInfo.PortalAddr

	aggPrioEqual := p.DRFNeighborAggregatorPriority == p.dr.DrniAggregatorPriority
	aggIdEqual := p.DRFNeighborAggregatorId == p.dr.DrniAggregatorId
	portalPrioEqual := p.DrniNeighborPortalPriority == p.dr.DrniPortalPriority
	portalAddrEqual := p.DrniNeighborPortalAddr == p.dr.DrniPortalAddr

	if aggPrioEqual &&
		aggIdEqual &&
		portalPrioEqual &&
		portalAddrEqual {
		p.DifferPortal = false
	} else {
		p.DifferPortal = true
		p.DifferPortalReason = ""
		if !aggPrioEqual {
			p.DifferPortalReason += "Neighbor Aggregator Priority, "
		}
		if !aggIdEqual {
			p.DifferPortalReason += "Neighbor Aggregator Id, "
		}
		if !portalPrioEqual {
			p.DifferPortalReason += "Neighbor Portal Priority, "
		}
		if !portalAddrEqual {
			p.DifferPortalReason += "Neighbor Portal Addr, "
		}
		// send the error to mgmt to correct the provisioning
		p.ReportToManagement()
	}
}

// recordPortalConfValues: 802.1ax-2014 Section 9.4.11 Functions
//
// This function records the Neighbor Portal System’s values carried in
// the Portal Configuration Information TLV of a received DRCPDU (9.4.3.2)
// from an IPP, as the current operational parameter values for the immediate
// Neighbor Portal System on this IPP as follows
func (rxm *RxMachine) recordPortalConfValues(drcpPduInfo *layers.DRCP) {
	p := rxm.p

	// Save config from pkt
	p.DRFNeighborPortalSystemNumber =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.TopologyState.GetState(layers.DRCPTopologyStatePortalSystemNum)
	p.DRFNeighborConfPortalSystemNumber =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.TopologyState.GetState(layers.DRCPTopologyStateNeighborConfPortalSystemNumber)
	p.DrniNeighborThreeSystemPortal =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.TopologyState.GetState(layers.DRCPTopologyState3SystemPortal) == 1
	p.DrniNeighborCommonMethods =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.TopologyState.GetState(layers.DRCPTopologyStateCommonMethods) == 1
	p.DrniNeighborONN =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.TopologyState.GetState(layers.DRCPTopologyStateOtherNonNeighbor) == 1
	p.DRFNeighborOperAggregatorKey =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.OperAggKey
	p.DRFNeighborPortAlgorithm =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.PortAlgorithm
	p.DRFNeighborConversationPortListDigest =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.PortDigest
	p.DRFNeighborGatewayAlgorithm =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.GatewayAlgorithm
	p.DRFNeighborConversationGatewayListDigest =
		drcpPduInfo.DRCPPortalConfigurationInfoTlv.GatewayDigest

	// which conversation vector is preset
	TwoPGatewayConverationVectorPresent := drcpPduInfo.TwoPortalGatewayConversationVector.TlvTypeLength.GetTlv() == layers.DRCPTLV2PGatewayConversationVector
	ThreePGatewayConversationVectorPresent := drcpPduInfo.ThreePortalGatewayConversationVector1.TlvTypeLength.GetTlv() == layers.DRCPTLV3PGatewayConversationVector1 &&
		drcpPduInfo.ThreePortalGatewayConversationVector2.TlvTypeLength.GetTlv() == layers.DRCPTLV3PGatewayConversationVector2
	TwoPPortConversationVectorPresent := drcpPduInfo.TwoPortalPortConversationVector.TlvTypeLength.GetTlv() == layers.DRCPTLV2PPortConversationVector
	ThreePPortConversationVectorPresent := drcpPduInfo.ThreePortalPortConversationVector1.TlvTypeLength.GetTlv() == layers.DRCPTLV3PPortConversationVector1 &&
		drcpPduInfo.ThreePortalPortConversationVector2.TlvTypeLength.GetTlv() == layers.DRCPTLV3PPortConversationVector2

	// It then compares the newly updated values of the Neighbor Portal System to this Portal
	// System’s expectations and if the comparison of
	portalSystemNumEqual := p.DRFNeighborPortalSystemNumber == p.DRFHomeConfNeighborPortalSystemNumber
	confPortalSystemNumEqual := p.DRFNeighborConfPortalSystemNumber == p.DRFPortalSystemNumber
	threeSystemPortalEqual := p.DrniNeighborThreeSystemPortal == p.d.DrniThreeSystemPortal
	commonMethodsEqual := p.DrniNeighborCommonMethods == p.d.DrniCommonMethods
	operAggKeyEqual := p.DRFNeighborOperAggregatoryKey[2:] == p.dr.DRFHomeOperAggregatorKey[2:]
	operAggKeyFullEqual := p.DRFNeighborOperAggregatoryKey == p.dr.DRFHomeOperAggregatorKey
	portAlgorithmEqual := p.DRFNeighborPortAlgorithm == p.dr.DRFHomePortAlgorithm
	conversationPortListDigestEqual := p.DRFNeighborConversationPortListDigest == p.dr.DRFHomeConverstaionPortListDigest
	gatewayAlgorithmEqual := p.DRFNeighborGatewayAlgorithm == p.dr.DRFHomeGatewayAlgorithm
	conversationGatewayListDigestEqual := p.DRFNeighborConversationGatewayListDigest == p.dr.DRFHomeConversationGatewayListDigest

	// lets set this as it will be cleared later if the fields differ
	p.DifferConfPortalSystemNumber = false
	p.ChangePortal = false
	p.MissingRcvGatewayConVector = true

	p.DifferPortalReason == ""
	if !portalSystemNumEqual {
		p.DifferConfPortalSystemNumber = true
		p.DifferPortalReason += "Portal System Number"
	}
	if !confPortalSystemNumEqual {
		p.DifferConfPortalSystemNumber = true
		p.DifferPortalReason += "Conf Portal System Number"
	}
	if operAggKeyEqual {
		p.DifferConfPortal = false
		if !operAggKeyFullEqual {
			p.ChangePortal = true
		}
	} else {
		p.DifferConfPortal = true
		p.DifferPortalReason += "Oper Aggregator Key"
	}
	/* TODO wording confusing in for this section so revisit later
	if !p.DifferConfPortal && (!threeSystemPortalEqual || gatewayAlgorithmEqual) {
		allNeighborConversation := [1024]uint8
		allBooleanVector := [1024]uint8
		for i := range allNeighborConversation {
			allNeighborConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber
			//allBooleanVector[i] =
		}
		//if !p.DifferConfPortal && p.DRNINeighborGatewayConversation ==  {

		//}

		if !threeSystemPortalEqual || {
			p.DifferPortalReason += "Three System Portal"
		}

		if !gatewayAlgorithmEqual {
			p.DifferPortalReason += "Gateway Algorithm"
		}
	}
	*/
	if !p.DifferConfPortal &&
		threeSystemPortalEqual &&
		gatewayAlgorithmEqual &&
		conversationGatewayListDigestEqual {
		p.DifferGatewayDigest = false
		p.GatewayConversationTransmit = false
		p.MissingRcvGatewayConVector = false
	}

	if !conversationGatewayListDigestEqual {
		p.DifferGatewayDigest = true
		p.GatewayConversationTransmit = true

		if TwoPGatewayConverationVectorPresent &&
			!p.dr.DrniThreeSystemPortal {
			i := 0
			// lets flatten out what comes in the packet to 4096 values
			for i := 0; i < 512; i++ {
				p.DrniNeighborGatewayConversation[i] = drcpPduInfo.TwoPortalGatewayConversationVector.Vector[i]
			}
			p.MissingRcvGatewayConVector = false
		}
		if ThreePGatewayConversationVectorPresent &&
			p.dr.DrniThreeSystemPortal {
			// concatinate the two 3P vector
			for i := 0; i < 512; i++ {
				p.DrniNeighborGatewayConversation[i] = drcpPduInfo.ThreePortalGatewayConversationVector1.Vector[i]

			}
			for i, j := 0, 512; i < 512; i, j = i+1, j+1 {
				p.DrniNeighborGatewayConversation[j] = drcpPduInfo.ThreePortalGatewayConversationVector2.Vector[i]
			}
			p.MissingRcvGatewayConVector = false
		}
	}

	if !TwoPGatewayConverationVectorPresent &&
		!ThreePGatewayConversationVectorPresent &&
		commonMethodsEqual &&
		p.dr.DrniCommonMethods &&
		(TwoPPortConversationVectorPresent ||
			ThreePPortConversationVectorPresent) {
		if TwoPPortConversationVectorPresent &&
			!p.dr.DrniThreeSystemPortal {
			// lets flatten out what comes in the packet to 4096 values
			for i := 0; i < 512; i++ {
				p.DrniNeighborGatewayConversation[i] = drcpPduInfo.TwoPortalPortConversationVector.Vector[i]
			}
			p.MissingRcvGatewayConVector = false
		} else if ThreePPortConversationVectorPresent &&
			p.dr.DrniThreeSystemPortal {
			// concatinate the two 3P vector
			for i := 0; i < 512; i++ {
				p.DrniNeighborGatewayConversation[i] = drcpPduInfo.ThreePortalPortConversationVector1.Vector[i]

			}
			for i, j := 0, 512; i < 512; i, j = i+1, j+1 {
				p.DrniNeighborGatewayConversation[j] = drcpPduInfo.ThreePortalPortConversationVector2.Vector[i]
			}
			p.MissingRcvGatewayConVector = false
		}
	}

	if !p.DifferConfPortal &&
		!threeSystemPortalEqual &&
		!portAlgorithmEqual {
		if conversationPortListDigestEqual {
			p.DifferPortDigest = false
			p.PortConversationTransmit = false
			p.MissingRcvPortConVector = false
		} else {
			p.DifferPortDigest = true
			p.PortConversationTransmit = true
			if TwoPPortConversationVectorPresent &&
				!p.dr.DrniThreeSystemPortal {
				if TwoPPortConversationVectorPresent &&
					!p.dr.DrniThreeSystemPortal {
					// lets flatten out what comes in the packet to 4096 values
					for i := 0; i < 512; i++ {
						p.DrniNeighborPortConversation[i] = drcpPduInfo.TwoPortalPortConversationVector.Vector[i]
					}
					p.MissingRcvPortConVector = false
				}
			} else if ThreePPortConversationVectorPresent &&
				p.dr.DrniThreeSystemPortal {
				// concatinate the two 3P vector
				for i := 0; i < 512; i++ {
					p.DrniNeighborPortConversation[i] = drcpPduInfo.ThreePortalPortConversationVector1.Vector[i]

				}
				for i, j := 0, 512; i < 512; i, j = i+1, j+1 {
					p.DrniNeighborPortConversation[j] = drcpPduInfo.ThreePortalPortConversationVector2.Vector[i]
				}
				p.MissingRcvGatewayConVector = false
			} else {
				/*
					LOGIC for this section does not make sense below based on the actions that are performed as they
					are covered above
					!TwoPPortConversationVectorPresent &&
					!ThreePPortConversationVectorPresent &&
					commonMethodsEqual &&
					p.dr.DrniCommonMethods {
					 Otherwise if no Port Conversation Vector TLVs (9.4.3.3.4, 9.4.3.3.5, 9.4.3.3.6) are
					present in the received DRCPDU, Drni_Neighbor_Common_Methods ==
					Drni_Common_Methods == TRUE OR the Port Conversation Vector TLVs
					(9.4.3.3.4, 9.4.3.3.5, 9.4.3.3.6) associated with the expected number of Portal
					Systems in the Portal are present in the received DRCPDU then;
					If aDrniThreeSystemPortal == FALSE,
					Drni_Neighbor_Port_Conversation = 2P_Gateway_Conversation_Vector;
					Otherwise if aDrniThreeSystemPortal == TRUE,
					Drni_Neighbor_Port_Conversation =
					concatenation
					of
					3P_Gateway_Conversation_Vector-1
					with
					3P_Gateway_Conversation_Vector-2;
					and;
					The variable Missing_Rcv_Port_Con_Vector is set to FALSE;
					Otherwise;
					The variable Drni_Neighbor_Port_Conversation remains unchanged and;
					The variable Missing_Rcv_Port_Con_Vector is set to TRUE.



					In all the operations above when the Drni_Neighbor_Gateway_Conversation or
					Drni_Neighbor_Port_Conversation are extracted from the Conversation Vector field in a
					received Conversation Vector TLV, the Drni_Neighbor_Gateway_Conversation or
					Drni_Neighbor_Port_Conversation will be set to All_Neighbor_Conversation, if
					Differ_Conf_Portal_System_Number == TRUE.
				*/
			}

		}
	}

	if p.DiffConfPortalSystemNumber {
		// set
		for i := 0; i < 1024; i++ {
			p.DrniNeighborGatewayConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber
			p.DrniNeighborPortConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber
		}
	}

	if !commonMethodsEqual {
		p.DifferPortalReason += "Common Methods"
	}
	if !portAlgorithmEqual {
		p.DifferPortalReason += "Port Algorithm"
	}
	if !conversationPortListDigestEqual {
		p.DifferPortalReason += "Converstaion Port List Digest"
	}
	if !conversationGatewayListDigestEqual {
		p.DifferPortalReason += "Conversation Gateway List Digest"
	}
}

// recordNeighborState: 802.1ax-2014 Section 9.4.11 Functions
//
// This function sets DRF_Neighbor_Oper_DRCP_State.IPP_Activity to TRUE and records the
//parameter values for the Drni_Portal_System_State[] and DRF_Home_Oper_DRCP_State
//carried in a received DRCPDU [item s) in 9.4.3.2] on the IPP, as the current parameter
// values for Drni_Neighbor_State[] and DRF_Neighbor_Oper_DRCP_State associated with
// this IPP respectively. In particular, the operational Boolean Gateway Vectors for
// each Portal System in the Drni_Neighbor_State[] are extracted from the received
// DRCPDU as follows
func (rxm *RxMachine) recordNeighborState(drcpPduInfo *layers.DRCP) {
	p := rxm.p

	neighborSystemNum := drcpPduInfo.DRCPPortalConfigurationInfoTlv.TopologyState.GetState(layers.DRCPTopologyStatePortalSystemNum)
	mySystemNum := p.dr.DrniPortalSystemNumber

	if neighborSystemNum != mySystemNum {

		if !drcpPduInfo.State.State.GetState(layers.DRCPStateHomeGatewayBit) {
			// clear all entries == NULL
			p.DRFRcvNeighborGatewayConversationMask = [MAX_GATEWAY_CONVERSATIONS]bool{}
		} else {
			if drcpPduInfo.HomeGatewayVector.DRCPTlvTypeLength.GetTlv() == layers.DRCPTLVTypeHomeGatewayVector {
				if len(drcpPduInfo.HomeGatewayVector.Vector) == 512 {
					p.DRFRcvNeighborGatewayConversationMask = drcpPduInfo.HomeGatewayVector.Vector
					p.DrniNeighborState[neighborSystemNum].updateNeighborVector(drcpPduInfo.State.Sequence, drcpPduInfo.State.Vector)
					// The OtherGatewayVectorTransmit on the other IPP, if it exists and is operational, is set to
					// TRUE
					// TODO is this the correct check to see if it exists and is operational??
					if drcpPduInfo.State.State.GetState(layers.DRCPStateOtherGatewayBit) {
						p.OtherGatewayVectorTransmit = true
					}
				} else if len(drcpPduInfo.HomeGatewayVector.Vector) == 0 {
					// The OtherGatewayVectorTransmit on the other IPP, if it exists and is operational, is set to
					// TRUE
					// TODO is this the correct check to see if it exists and is operational??
					if drcpPduInfo.State.State.GetState(layers.DRCPStateOtherGatewayBit) {
						p.OtherGatewayVectorTransmit = false
						index := p.getNeighborVectorGatwaySequenceIndex(drcpPduInfo.HomeGatewayVector.Sequence,
							drcpPduInfo.HomeGatewayVector.Vector)
						if index == 0 {
							for i, j := 0, 0; i < 512; i, j = i+1, j+4 {
								if p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]>>3&0x1 == 1 {
									p.DRFRcvNeighborGatewayConversationMask[j] = true
								}
								if p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]>>2&0x1 == 1 {
									p.DRFRcvNeighborGatewayConversationMask[j+1] = true
								}
								if p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]>>1&0x1 == 1 {
									p.DRFRcvNeighborGatewayConversationMask[j+2] = true
								}
								if p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]>>0&0x1 == 1 {
									p.DRFRcvNeighborGatewayConversationMask[j+3] = true
								}
							}
						} else {
							for i := 0; i < 4096; i++ {
								p.DRFRcvNeighborGatewayConversationMask[i] = true
							}
						}
					}
				}
			}
		}
		if !drcpPduInfo.State.State.GetState(layers.DRCPStateNeighborGatewayBit) {
			// clear all entries == NULL
			p.DRFRcvHomeGatewayConversationMask = [MAX_GATEWAY_CONVERSATIONS]bool{}
		} else {
			index := p.getNeighborVectorGatwaySequenceIndex(drcpPduInfo.NeighborGatewayVector.Sequence,
				drcpPduInfo.NeighborGatewayVector.Vector)
			if index != -1 {
				for i, j := 0, 0; i < 512; i, j = i+1, j+4 {
					if p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]>>3&0x1 == 1 {
						p.DRFRcvHomeGatewayConversationMask[j] = true
					}
					if p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]>>2&0x1 == 1 {
						p.DRFRcvHomeGatewayConversationMask[j+1] = true
					}
					if p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]>>1&0x1 == 1 {
						p.DRFRcvHomeGatewayConversationMask[j+2] = true
					}
					if p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector[i]>>0&0x1 == 1 {
						p.DRFRcvHomeGatewayConversationMask[j+3] = true
					}
				}
				if index == 0 {
					p.HomeGatewayVectorTransmit = false
				} else {
					p.HomeGatewayVectorTransmit = true
					if drcpPduInfo.NeighborGatewayVector.Sequence > p.DRFRcvHomeGatewaySequence {
						p.DrniNeighborState[neighborSystemNum].updateNeighborVector(drcpPduInfo.NeighborGatewayVector.Sequence+1,
							p.DrniNeighborState[neighborSystemNum].GatewayVector[index].Vector)
					}
				}
			} else {
				for i := 0; i < 4096; i++ {
					p.DRFRcvNeighborGatewayConversationMask[i] = true
				}
				p.HomeGatewayVectorTransmit = true
			}
		}
		if !drcpPduInfo.State.State.GetState(layers.DRCPStateOtherGatewayBit) {
			// clear all entries == NULL
			p.DRFRcvOtherGatewayConversationMask = [MAX_GATEWAY_CONVERSATIONS]bool{}
		} else {
			if drcpPduInfo.OtherGatewayVector.TlvTypeLength.GetTlv() == layers.DRCPTLVTypeOtherGatewayVector {
				if len(drcpPduInfo.OtherGatewayVector.Vector) > 0 {
					p.DRFRcvOtherGatewayConversationMask = drcpPduInfo.OtherGatewayVector.Vector
					if !p.DRNINeighborONN {
						p.DrniNeighborState[neighborSystemNum].updateNeighborVector(drcpPduInfo.OtherGatewayVector.Sequence,
							drcpPduInfo.OtherGatewayVecto.Vector)
					}
				} else {
					index := p.getNeighborVectorGatwaySequenceIndex(drcpPduInfo.OtherGatewayVector.Sequence,
						drcpPduInfo.NeighborGatewayVector.Vector)
					if index != -1 {
						p.DRFRcvOtherGatewayConversationMask = p.DRFRcvNeighborGatewayConversationMask[index]
						if index == 1 {
							p.OtherGatewayVectorTransmit = false
						} else {
							p.OtherGatewayVectorTransmit = true
						}
					} else {
						for i := 0; i < 4096; i++ {
							p.DRFRcvOtherGatewayConversationMask[i] = true
						}
						p.OtherGatewayVectorTransmit = true
					}
				}
			}
		}

		// Lets also record the following variables
		if drcpPduInfo.State.State.GetState(layers.DRCPStateHomeGatewayBit) {
			p.DRFHomeOperDRCPState.SetState(layers.DRCPStateHomeGatewayBit)
		} else {
			p.DRFHomeOperDRCPState.ClearState(layers.DRCPStateHomeGatewayBit)
		}
		if drcpPduInfo.HomePortsInfo.TlvTypeLength.GetTlv() == layers.DRCPTLVTypeHomePortsInfo {
			// Active_Home_Ports in the Home Ports Information TLV, carried in a
			// received DRCPDU on the IPP, are used as the current values for the DRF_Neighbor_State on
			// this IPP and are associated with the Portal System identified by DRF_Neighbor_Portal_System_Number;
			p.DrniNeighborState[neighborSystemNum].PortIdList = drcpPduInfo.HomePortsInfo.ActiveNeighborPorts
		}
		// TODO the Gateway Vector from the most recent entry in the Gateway Vector database for the Neighbor Portal System

		if drcpPduInfo.State.State.GetState(layers.DRCPStateOtherGatewayBit) {
			p.DRFHomeOperDRCPState.SetState(layers.DRCPStateOtherGatewayBit)
		} else {
			p.DRFHomeOperDRCPState.ClearState(layers.DRCPStateOtherGatewayBit)
		}

		p.compareOtherPortsInfo(drcpPduInfo)

		// Network / IPL sharing by time (9.3.2.1) is supported
		p.compareNetworkIPLMethod(drcpPduInfo)

		// TODO Further, if Network / IPL sharing by tag (9.3.2.2) or Network / IPL sharing by encapsulation
		// (9.3.2.3) is supported
		p.compareGatewayOperGatewayVector()
		p.comparePortIds()
	}
}

func (rxm *RxMachine) compareOtherPortsInfo(drcpPduInfo *layers.DRCP) {
	p := rxm.p
	if drcpPduInfo.OtherPortsInfo.TlvTypeLength.GetTlv() == layers.DRCPTLVTypeOtherPortsInfo {
		// the Other_Neighbor_Ports in the Other Ports Information TLV,
		// carried in a received DRCPDU on the IPP, are used as the current values for the
		// DRF_Other_Neighbor_State on this IPP and are associated with the Portal System identified
		// by the value assigned to the two most significant bits of the
		// DRF_Other_Neighbor_Admin_Aggregator_Key carried within the Other Ports Information
		// TLV in the received DRCPDU
		p.DRFOtherNeighborState.PortIdList = drcpPduInfo.OtherPortsInfo.NeighborPorts
	} else if p.dr.DrniThreeSystemPortal {
		p.DRFOtherNeighborState.OpState = false
		p.DRFOtherNeighborState.GatewayVector = nil
		p.DRFOtherNeighborState.PortIdList = nil
		// no Portal System state information is available on this IPP for the distant
		// Neighbor Portal System on the IPP
		// TODO what does this mean that the info is not present for now going to clear/set
		// the keys
		p.DRFNeighborAdminAggregatorKey = p.dr.DRFHomeAdminAggregatorKey
		p.DRFNeighborOperPartnerAggregatorKey = p.dr.DRFHomeOperPartnerAggregatorKey
		// TODO Document states to set to this but if we reached this if the other should be
		// set to NULL
		//p.DRFOtherNeighborAdminAggregatorKey = p.DRFOtherNeighborAdminAggregatorKey
		//p.DRFOtherNeighborOperPartnerAggregatorKey = p.DRFOtherNeighborOperPartnerAggregatorKey
		p.DRFOtherNeighborAdminAggregatorKey = 0
		p.DRFOtherNeighborOperPartnerAggregatorKey = 0
	}
}

func (rxm *RxMachine) compareNetworkIPLMethod(drcpPduInfo *layers.DRCP) {
	p := rxm.p
	if drcpPduInfo.NetworkIPLMethod.TlvTypeLength.GetTlv() == layers.DRCPTLVNetworkIPLSharingMethod {
		p.DRFNeighborNetworkIPLSharingMethod = drcpPduInfo.NetworkIPLMethod.Method
		if p.DRFNeighborNetworkIPLSharingMethod == p.DRFHomeNetworkIPLSharingMethod {
			p.CCTimeShared = true
		} else {
			p.CCTimeShared = false
		}
	}
}

func (rxm *RxMachine) compareGatewayOperGatewayVector() {

	p := rxm.p

	operOrVectorDiffer := false
	for i := 0; i < 3 && !operOrVectorDiffer; i++ {
		if p.DrniPortalSystemState[i].OpState != p.DrniNeighborSystemState[i].OpState ||
			p.DrniPortalSystemState[i].GatewayVector != p.DrniNeighborSystemState[i].GatewayVector {
			operOrVectorDiffer = true
		}
	}

	if operOrVectorDiffer {
		p.dr.GatewayConversationUpdate = true
		p.DRFHomeOperDRCPState.ClearState(layers.DRCPStateGatewaySync)
		if p.MissingRcvGatewayConVector {
			for i := 0; i < 1024; i++ {
				if p.dr.DrniThreeSystemPortal {
					p.DrniNeighborGatewayConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber
				} else {
					p.DrniNeighborGatewayConversation[i] = 0xff
				}
			}
		}
	} else {
		if !p.DRFHomeOperDRCPState.GetState(layers.DRCPStateGatewaySync) {
			p.dr.GatewayConversationUpdate = true
			p.DRFHomeOperDRCPState.SetState(layers.DRCPStateGatewaySync)
		} else if p.DifferGatewayDigest {
			p.dr.GatewayConversationUpdate = true
		} else {
			p.DRFHomeOperDRCPState.SetState(layers.DRCPStateGatewaySync)
		}
	}
}

// Helper Functions For sort
func (s sortPortList) Len() int           { return len(s) }
func (s sortPortList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sortPortList) Less(i, j int) bool { return s[i] < s[j] }

func (rxm *RxMachine) comparePortIds() {
	p := rxm.p
	portListDiffer := false

	for i := 0; i < 3 && !operOrVectorDiffer; i++ {
		if len(p.DrniPortalSystemState[i].PortIdList) == len(p.DrniNeighborSystemState[i].PortIdList) {
			// make sure the lists are sorted
			sort.Sort(sortPortList(p.DrniPortalSystemState[i].PortIdList))
			sort.Sort(sortPortList(p.DrniNeighborSystemState[i].PortIdList))
			if p.DrniPortalSystemState[i].PortIdList != p.DrniNeighborSystemState[i].PortIdList {
				portListDiffer = true
			}
		} else {
			portListDiffer = true
		}
	}

	if portListDiffer {
		p.dr.PortConversationUpdate = true
		p.DRFHomeOperDRCPState.ClearState(layers.DRCPStatePortSync)
		if p.MissingRcvPortConVector {
			for i := 0; i < 1024; i++ {
				if p.dr.DrniThreeSystemPortal {
					p.DrniNeighborPortConversation[i] = p.DRFHomeConfNeighborPortalSystemNumber
				} else {
					p.DrniNeighborPortConversation[i] = 0xff
				}
			}
		}
	} else {
		if p.DifferPortDigest {
			p.dr.PortConversationUpdate = true
		} else if !p.DRFHomeOperDRCPState.GetState(layers.DRCPStatePortSync) {
			p.dr.PortConversationUpdate = true
			p.DRFHomeOperDRCPState.SetState(layers.DRCPStatePortSync)
		} else {
			p.DRFHomeOperDRCPState.SetState(layers.DRCPStatePortSync)
		}
	}
}
