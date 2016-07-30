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
// 802.1ax-2014 Section 9.4.15 DRCPDU Periodic Transmission machine
// rxmachine.go
package drcp

import (
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/utils"
	"sort"
	"time"
)

const PsMachineModuleStr = "Portal System Machine"

// psm States
const (
	PsmStateNone = iota + 1
	PsmStatePortalSystemInitialize
	PsmStatePortalSystemUpdate
)

var PsmStateStrMap map[fsm.State]string

func PsMachineStrStateMapCreate() {
	PsmStateStrMap = make(map[fsm.State]string)
	PsmStateStrMap[PsmStateNone] = "None"
	PsmStateStrMap[PsmStatePortalSystemInitialize] = "Portal System Initialize"
	PsmStateStrMap[PsmStatePortalSystemUpdate] = "Portal System Update"
}

// psm events
const (
	PsmEventBegin = iota + 1
	PsmEventChangePortal
	PsmEventChangeDRFPorts
)

// PsMachine holds FSM and current State
// and event channels for State transitions
type PsMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	dr *DistributedRelay

	// machine specific events
	PsmEvents chan utils.MachineEvent
}

// PrevState will get the previous State
func (psm *PsMachine) PrevState() fsm.State { return psm.PreviousState }

// PrevStateSet will set the previous State
func (psm *PsMachine) PrevStateSet(s fsm.State) { psm.PreviousState = s }

// Stop should clean up all resources
func (psm *PsMachine) Stop() {
	close(psm.PsmEvents)
}

// NewDrcpPsMachine will create a new instance of the PsMachine
func NewDrcpPsMachine(dr *DistributedRelay) *PsMachine {
	psm := &PsMachine{
		dr:            dr,
		PreviousState: PsmStateNone,
		PsmEvents:     make(chan MachineEvent, 10),
	}

	dr.PsMachineFsm = psm

	return psm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (psm *PsMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if psm.Machine == nil {
		psm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	psm.Machine.Rules = r
	psm.Machine.Curr = &utils.StateEvent{
		StrStateMap: PsmStateStrMap,
		LogEna:      true,
		Logger:      psm.DrcpPsmLog,
		Owner:       PsMachineModuleStr,
	}

	return psm.Machine
}

// DrcpPsMachinePortalSystemInitialize function to be called after
// State transition to PORTAL_SYSTEM_INITIALIZE
func (psm *PsMachine) DrcpPsMachinePortalSystemInitialize(m fsm.Machine, data interface{}) fsm.State {
	psm.setDefaultPortalSystemParameters()
	psm.updateKey()
	return PsmStatePortalSystemInitialize
}

// DrcpPsMachineFastPeriodic function to be called after
// State transition to FAST_PERIODIC
func (psm *PsMachine) DrcpPsMachinePortalSystemUpdate(m fsm.Machine, data interface{}) fsm.State {
	dr := psm.dr

	dr.ChangePortal = false
	dr.ChangeDRFPorts = false
	psm.updateDRFHomeState()
	psm.updateKey()

	// next State
	return PsmStatePortalSystemUpdate
}

func DrcpPsMachineFSMBuild(p *DRCPIpp) *PsMachine {

	PsMachineStrStateMapCreate()

	rules := fsm.Ruleset{}

	// Instantiate a new PsMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the initalize State
	psm := NewDrcpPsMachine(p)

	//BEGIN -> PORTAL SYSTEM INITITIALIZE
	rules.AddRule(PsmStateNone, PsmEventBegin, psm.DrcpPsMachinePortalSystemInitialize)
	rules.AddRule(PsmStatePortalSystemInitialize, PsmEventBegin, psm.DrcpPsMachinePortalSystemInitialize)
	rules.AddRule(PsmStatePortalSystemUpdate, PsmEventBegin, psm.DrcpPsMachinePortalSystemInitialize)

	// CHANGE PORTAL  > PORTAL SYSTEM UPDATE
	rules.AddRule(PsmStatePortalSystemInitialize, PsmEventChangePortal, psm.DrcpPsMachinePortalSystemUpdate)
	rules.AddRule(PsmStatePortalSystemUpdate, PsmEventChangePortal, psm.DrcpPsMachinePortalSystemUpdate)

	// CHANGE DRF PORTS  > PORTAL SYSTEM UPDATE
	rules.AddRule(PsmStatePortalSystemInitialize, PsmEventChangeDRFPorts, psm.DrcpPsMachinePortalSystemUpdate)
	rules.AddRule(PsmStatePortalSystemUpdate, PsmEventChangeDRFPorts, psm.DrcpPsMachinePortalSystemUpdate)

	// Create a new FSM and apply the rules
	rxm.Apply(&rules)

	return rxm
}

// DrcpPsMachineMain:  802.1ax-2014 Figure 9-25
// Creation of Portal System Machine State transitions and callbacks
// and create go routine to pend on events
func (p *DRCPIpp) DrcpPsMachineMain() {

	// Build the State machine for  DRCP Portal System Machine according to
	// 802.1ax-2014 Section 9.4.16 Portal System Machine
	psm := DrcpPsMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	psm.Machine.Start(psm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the PsMachine should handle.
	go func(m *PsMachine) {
		m.DrcpPsmLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case event, ok := <-m.PsmEvents:
				if ok {
					rv := m.Machine.ProcessEvent(event.src, event.e, nil)
					if rv == nil {
						p := m.p
						// port state processing
						if p.ChangePortal {
							rv = m.Machine.ProcessEvent(PsMachineModuleStr, PsmEventChangePortal, nil)
						} else if p.ChangeDRFPorts {
							rv = m.Machine.ProcessEvent(PsMachineModuleStr, PsmEventChangeDRFPorts, nil)
						}
					}

					if rv != nil {
						m.DrcpGmLog(strings.Join([]string{error.Error(rv), event.src, PsmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
					}
				}

				// respond to caller if necessary so that we don't have a deadlock
				if event.ResponseChan != nil {
					utils.SendResponse(PsMachineModuleStr, event.ResponseChan)
				}
				if !ok {
					m.DrcpPsmLog("Machine End")
					return
				}
			}
		}
	}(psm)
}

// setDefaultPortalSystemParameters This function sets this Portal System’s variables
// to administrative set values as follows
func (psm PsMachine) setDefaultPortalSystemParameters() {
	dr := psm.dr
	a := dr.a

	dr.DrniAggregatorPriority = a.AggSystemPriority
	dr.DrniAggregatorID = a.AggMacAddr
	dr.DrniPortalPriority = dr.DrniPortalPriority
	dr.DrniPortalAddr = dr.DrniPortalAddr
	dr.DRFPortalSystemNumber = dr.DrniPortalSystemNumber
	dr.DRFHomeAdminAggregator = a.ActorAdminKey
	dr.DRFHomePortAlgorithm = a.PortAlgorithm
	dr.DRFHomeGatewayAlgorithm = dr.DrniGatewayAlgorithm
	dr.DRFHomeOperDRCPState = dr.DRFNeighborAdminDRCPState
	for i := 0; i < 3; i++ {
		dr.DrniPortalSystemState[0].OpState = false
		dr.GatewayVector = nil
		dr.PortIdList = nil
	}

	// TOOD
	//dr.DRFHomeConversationPortListDigest
	//dr.DRFHomeConversationGatewayListDigest

}

//updateKey This function updates the operational Aggregator Key,
//DRF_Home_Oper_Aggregator_Key, as follows
func (psm *PsMachine) updateKey() {
	dr := psm.dr
	a := dr.a

	if a != nil &&
		a.PartnerDWC {
		dr.DRFHomeOperAggregatorKey = ((dr.DRFHomeAdminAggregatorKey && 0x3fff) | 0x6000)
	} else if dr.PSI &&
		!p.DRFHomeOperDRCPState.GetState(layers.DRCPStateHomeGatewayBit) {
		dr.DRFHomeOperAggregatorKey = (dr.DRFHomeAdminAggregatorKey && 0x3fff)
	} else {
		// lets get the lowest admin key and set this to the
		// operational key
		if dr.DRFHomeAdminAggregatorKey != 0 &&
			dr.DRFHomeAdminAggregatorKey < dr.DRFNeighborAdminAggregatorKey &&
			dr.DRFHomeAdminAggregatorKey < dr.DRFOtherNeighborAdminAggregatoryKey {
			dr.DRFHomeOperAggregatorKey = dr.DRFHomeAdminAggregatorKey
		} else if dr.DRFNeighborAdminAggregatorKey != 0 &&
			dr.DRFNeighborAdminAggregatorKey < dr.DRFHomeAdminAggregatorKey &&
			dr.DRFNeighborAdminAggregatorKey < dr.DRFOtherNeighborAdminAggregatoryKey {
			dr.DRFHomeOperAggregatorKey = dr.DRFNeighborAdminAggregatorKey
		} else if dr.DRFOtherNeighborAdminAggregatoryKey != 0 &&
			dr.DRFOtherNeighborAdminAggregatoryKey < dr.DRFHomeAdminAggregatorKey &&
			dr.DRFOtherNeighborAdminAggregatoryKey < dr.DRFNeighborAdminAggregatorKey {
			dr.DRFHomeOperAggregatorKey = dr.DRFOtherNeighborAdminAggregatoryKey
		}
	}
}

//updateDRFHomeState This function updates the DRF_Home_State based on the operational
//state of the local ports as follows
func (psm *PsMachine) updateDRFHomeState() {
	// TODO need to understand the logic better
}
