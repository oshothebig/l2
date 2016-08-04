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
	"strconv"
	"strings"
	"utils/fsm"
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
		PsmEvents:     make(chan utils.MachineEvent, 10),
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

func DrcpPsMachineFSMBuild(dr *DistributedRelay) *PsMachine {

	PsMachineStrStateMapCreate()

	rules := fsm.Ruleset{}

	// Instantiate a new PsMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the initalize State
	psm := NewDrcpPsMachine(dr)

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
	psm.Apply(&rules)

	return psm
}

// DrcpPsMachineMain:  802.1ax-2014 Figure 9-25
// Creation of Portal System Machine State transitions and callbacks
// and create go routine to pend on events
func (dr *DistributedRelay) DrcpPsMachineMain() {

	// Build the State machine for  DRCP Portal System Machine according to
	// 802.1ax-2014 Section 9.4.16 Portal System Machine
	psm := DrcpPsMachineFSMBuild(dr)
	//dr.wg.Add(1)
	dr.waitgroupadd("PSM")

	// set the inital State
	psm.Machine.Start(psm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the PsMachine should handle.
	go func(m *PsMachine) {
		m.DrcpPsmLog("Machine Start")
		//defer m.dr.wg.Done()
		defer m.dr.waitgroupstop("PSM")
		for {
			select {
			case event, ok := <-m.PsmEvents:
				if ok {
					rv := m.Machine.ProcessEvent(event.Src, event.E, nil)
					if rv == nil {
						dr := m.dr
						// port state processing
						if dr.ChangePortal {
							rv = m.Machine.ProcessEvent(PsMachineModuleStr, PsmEventChangePortal, nil)
						} else if dr.ChangeDRFPorts {
							rv = m.Machine.ProcessEvent(PsMachineModuleStr, PsmEventChangeDRFPorts, nil)
						}
					}

					if rv != nil {
						m.DrcpPsmLog(strings.Join([]string{error.Error(rv), event.Src, PsmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.E))}, ":"))
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

	dr.DrniAggregatorPriority = a.AggPriority
	dr.DrniAggregatorId = a.AggMacAddr
	dr.DrniPortalPriority = dr.DrniPortalPriority
	dr.DrniPortalAddr = dr.DrniPortalAddr
	dr.DRFPortalSystemNumber = dr.DrniPortalSystemNumber
	dr.DRFHomeAdminAggregatorKey = a.ActorAdminKey
	dr.DRFHomePortAlgorithm = a.PortAlgorithm
	dr.DRFHomeGatewayAlgorithm = dr.DrniGatewayAlgorithm
	dr.DRFHomeOperDRCPState = dr.DRFNeighborAdminDRCPState
	dr.GatewayVectorDatabase = nil
	// set during config do not want to clear this to a default because
	// there is only one default
	// dr.Ipplinks = nil
	for i := 0; i < 3; i++ {
		dr.DrniPortalSystemState[0].OpState = false
		// Don't want to clear this as it is set by config
		//dr.DrniIntraPortalLinkList[i] = 0
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
		dr.DRFHomeOperAggregatorKey = ((dr.DRFHomeAdminAggregatorKey & 0x3fff) | 0x6000)
	} else if dr.PSI &&
		!dr.DRFHomeOperDRCPState.GetState(layers.DRCPStateHomeGatewayBit) {
		dr.DRFHomeOperAggregatorKey = (dr.DRFHomeAdminAggregatorKey & 0x3fff)
	} else {
		// lets get the lowest admin key and set this to the
		// operational key
		var operKey uint16
		for _, ipp := range dr.Ipplinks {

			if dr.DRFHomeAdminAggregatorKey != 0 &&
				dr.DRFHomeAdminAggregatorKey < ipp.DRFNeighborAdminAggregatorKey &&
				dr.DRFHomeAdminAggregatorKey < ipp.DRFOtherNeighborAdminAggregatorKey {
				if operKey == 0 || dr.DRFHomeAdminAggregatorKey < operKey {
					operKey = dr.DRFHomeAdminAggregatorKey
				}
			} else if ipp.DRFNeighborAdminAggregatorKey != 0 &&
				ipp.DRFNeighborAdminAggregatorKey < dr.DRFHomeAdminAggregatorKey &&
				ipp.DRFNeighborAdminAggregatorKey < ipp.DRFOtherNeighborAdminAggregatorKey {
				if operKey == 0 || ipp.DRFNeighborAdminAggregatorKey < operKey {
					operKey = ipp.DRFNeighborAdminAggregatorKey
				}
			} else if ipp.DRFOtherNeighborAdminAggregatorKey != 0 &&
				ipp.DRFOtherNeighborAdminAggregatorKey < dr.DRFHomeAdminAggregatorKey &&
				ipp.DRFOtherNeighborAdminAggregatorKey < ipp.DRFNeighborAdminAggregatorKey {
				if operKey == 0 || ipp.DRFOtherNeighborAdminAggregatorKey < operKey {
					operKey = ipp.DRFOtherNeighborAdminAggregatorKey
				}
			}
		}
		dr.DRFHomeOperAggregatorKey = operKey
	}
}

//updateDRFHomeState This function updates the DRF_Home_State based on the operational
//state of the local ports as follows
func (psm *PsMachine) updateDRFHomeState() {
	// TODO need to understand the logic better
}
