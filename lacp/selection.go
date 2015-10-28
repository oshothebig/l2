// selection
package lacp

import ()

/*
Aggregation is represented by an Aggregation Port selecting an appropriate Aggregator, and then attaching
to that Aggregator. The following are required for correct operation of the selection and attachment logic:

a) The implementation shall support at least one Aggregator per System.
b) Each Aggregation Port shall be assigned an operational Key (6.3.5). Aggregation Ports that can
   aggregate together are assigned the same operational Key as the other Aggregation Ports with which
   they can aggregate; Aggregation Ports that cannot aggregate with any other Aggregation Port are
   allocated unique operational Keys.
c) Each Aggregator shall be assigned an operational Key.
d) Each Aggregator shall be assigned an identifier that distinguishes it among the set of Aggregators in
   the System.
e) An Aggregation Port shall only select an Aggregator that has the same operational Key assignment
   as its own operational Key.
f) Subject to the exception stated in item g), Aggregation Ports that are members of the same LAG
   (i.e., two or more Aggregation Ports that have the same Actor System ID, Actor Key, Partner System
   ID, and Partner Key, and that are not required to be Individual) shall select the same Aggregator.
g) Any pair of Aggregation Ports that are members of the same LAG, but are connected together by the
   same link, shall not select the same Aggregator (i.e., if a loopback condition exists between two
   Aggregation Ports, they shall not be aggregated together. For both Aggregation Ports, the Actor
   System ID is the same as the Partner System ID; also, for Aggregation Port A, the Partner’s Port
   Identifier is Aggregation Port B, and for Aggregation Port B, the Partner’s Port Identifier is
   Aggregation Port A).
   NOTE 1—This exception condition prevents the formation of an aggregated link, comprising two ends of the
   same link aggregated together, in which all frames transmitted through an Aggregator are immediately received
   through the same Aggregator. However, it permits the aggregation of multiple links that are in loopback; for
   example, if Aggregation Port A is looped back to Aggregation Port C and Aggregation Port B is looped back to
   Aggregation Port D, then it is permissible for A and B (or A and D) to aggregate together, and for C and D (or
   B and C) to aggregate together.
h) Any Aggregation Port that is required to be Individual (i.e., the operational state for the Actor or the
   Partner indicates that the Aggregation Port is Individual) shall not select the same Aggregator as any
   other Aggregation Port.
i) Any Aggregation Port that is Aggregateable shall not select an Aggregator to which an Individual
   Aggregation Port is already attached.
j) If the preceding conditions result in a given Aggregation Port being unable to select an Aggregator,
   then that Aggregation Port shall not be attached to any Aggregator.
k) If there are further constraints on the attachment of Aggregation Ports that have selected an
   Aggregator, those Aggregation Ports may be selected as standby in accordance with the rules
   specified in 6.7.1. Selection or deselection of that Aggregator can cause the Selection Logic to
   reevaluate the Aggregation Ports to be selected as standby.
l) The Selection Logic operates upon the operational information recorded by the Receive state
   machine, along with knowledge of the Actor’s own operational configuration and state. The
   Selection Logic uses the LAG ID for the Aggregation Port, determined from these operational
   parameters, to locate the correct Aggregator to which to attach the Aggregation Port.
m) The Selection Logic is invoked whenever an Aggregation Port is not attached to and has not selected
   an Aggregator, and executes continuously until it has determined the correct Aggregator for the
   Aggregation Port.
   NOTE 2—The Selection Logic may take a significant time to complete its determination of the correct
   Aggregator, as a suitable Aggregator may not be immediately available, due to configuration restrictions or the
   time taken to reallocate Aggregation Ports to other Aggregators.
n) Once the correct Aggregator has been determined, the variable Selected shall be set to SELECTED
   or to STANDBY (6.4.8, 6.7.1).
   NOTE 3—If Selected is SELECTED, the Mux machine will start the process of attaching the Aggregation Port
   to the selected Aggregator. If Selected is STANDBY, the Mux machine holds the Aggregation Port in the
   WAITING state, ready to be attached to its Aggregator once its Selected state changes to SELECTED.
o) The Selection Logic is responsible for computing the value of the Ready variable from the values of
   the Ready_N variable(s) associated with the set of Aggregation Ports that are waiting to attach to the
   same Aggregator (see 6.4.8).
p) Where the selection of a new Aggregator by an Aggregation Port, as a result of changes to the
   selection parameters, results in other Aggregation Ports in the System being required to reselect their
   Aggregators in turn, this is achieved by setting Selected to UNSELECTED for those other
   Aggregation Ports that are required to reselect their Aggregators.
   NOTE 4—The value of Selected is set to UNSELECTED by the Receive machine for the Aggregation Port
   when a change of LAG ID is detected.
q) An Aggregation Port shall not be enabled for use by the Aggregator Client until it has both selected
   and attached to an Aggregator.
r) An Aggregation Port shall not select an Aggregator, which has been assigned to a Portal (Clause 9),
   unless the Partner_Oper_Key of the associated LAG ID is equal to the lowest numerical value of the
   set comprising the values of the DRF_Home_Oper_Partner_Aggregator_Key, the
   DRF_Neighbor_Oper_Partner_Aggregator_Key,
   and the DRF_Other_Neighbor_Oper_Partner_Aggregator_Key, on each IPP on the Portal System, where
   any variable having its most significant bits set to 00 is excluded (corresponding to accepting only
   variables that have an integer value that is larger than 16383).
s) An Aggregation Port shall not select an Aggregator, which has been assigned to a Portal (Clause 9),
   if its Portal’s System Identifier is set to a value that is numerically lower than the Partner’s System
   Identifier, PSI == TRUE, the most significant two bits of Partner_Oper_Key are equal to the value 2
   or 3 and the two least significant bits of the Aggregation Port’s Partner_Oper_Port_Priority are
   equal to the value 2 or 3. This is to prevent network partition due to isolation of the Portal Systems
   in the interconnected Portals (Clause 9)
*/
/*
func (p *LaAggPort) LacpSelectAggrigator() bool {

	p.aggId
	if _, ok := gLacpSysGlobalInfo.AggMap[p.aggId] ok {

	}
	else {

	}

}
*/
// LacpMuxCheckSelectionLogic will be called after the
// wait while timer has expired.  If this is the last
// port to have its wait while timer expire then
// will transition the mux state from waiting to
// attached
func (agg *LaAggregator) LacpMuxCheckSelectionLogic(p *LaAggPort) {

	readyChan := make(chan bool)
	// lets do a this work in parrallel
	for _, pId := range agg.PortNumList {

		go func(id uint16) {
			var port *LaAggPort
			if LaFindPortById(id, &port) {
				readyChan <- port.readyN
			}
		}(pId)
	}
	close(readyChan)

	// lets set the agg.ready flag to true
	// until we know that at least one port
	// is not ready
	agg.ready = true
	for range agg.PortNumList {
		select {
		case readyN := <-readyChan:
			if !readyN {
				agg.ready = false
			}
		}
	}

	// if agg is ready then lets attach the
	// ports which are not already attached
	if agg.ready {
		// lets do this work in parrallel
		for _, pId := range agg.PortNumList {
			go func(id uint16) {
				var port *LaAggPort
				if LaFindPortById(id, &port) &&
					port.readyN &&
					port.aggAttached != nil {
					// trigger event to mux
					port.MuxMachineFsm.MuxmEvents <- LacpMachineEvent{e: LacpMuxmEventSelectedEqualSelectedAndReady,
						src: MuxMachineModuleStr}
				}
			}(pId)
		}
	}
}

// updateSelected:  802.1ax Section 6.4.9
// Sets the value of the Selected variable based on the following:
//
// Rx pdu: (Actor: Port, Priority, System, System Priority, Key
// and State Aggregation) vs (Partner Oper: Port, Priority,
// System, System Priority, Key, State Aggregation).  If values
// have changed then Selected is set to UNSELECTED, othewise
// SELECTED
func (rxm *LacpRxMachine) updateSelected(lacpPduInfo *LacpPdu) {

	p := rxm.p

	if !LacpLacpPortInfoIsEqual(&lacpPduInfo.actor.info, &p.partnerOper, LacpStateAggregationBit) {

		rxm.LacpRxmLog("PDU and Oper states do not agree, moving port to LacpAggUnSelected")

		// lets trigger the event only if mux is not in waiting state as
		// the wait while timer expiration will trigger the unselected event
		if p.MuxMachineFsm != nil &&
			p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateWaiting &&
			p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateWaiting {
			p.aggSelected = LacpAggUnSelected
			p.MuxMachineFsm.MuxmEvents <- LacpMachineEvent{e: LacpMuxmEventSelectedEqualUnselected}
		}
	}
}

// updateDefaultedSelected: 802.1ax Section 6.4.9
//
// Update the value of the Selected variable comparing
// the Partner admin info based with the partner
// operational info
// (port num, port priority, system, system priority,
//  key, stat.Aggregation)
func (rxm *LacpRxMachine) updateDefaultSelected() {

	p := rxm.p

	if !LacpLacpPortInfoIsEqual(&p.partnerAdmin, &p.partnerOper, LacpStateAggregationBit) {

		//p.aggSelected = LacpAggUnSelected
		// lets trigger the event only if mux is not in waiting state as
		// the wait while timer expiration will trigger the unselected event
		if p.MuxMachineFsm != nil &&
			p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateWaiting &&
			p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateWaiting {
			p.aggSelected = LacpAggUnSelected
			p.MuxMachineFsm.MuxmEvents <- LacpMachineEvent{e: LacpMuxmEventSelectedEqualUnselected}
		}

	}
}

// checkConfigForSelection will send selection bit to state machine
// and return to the user true
func (p *LaAggPort) checkConfigForSelection() bool {
	var a LaAggregator

	// check to see if aggrigator exists
	// and that the keys match
	if p.aggId != 0 && LaFindAggById(p.aggId, &a) &&
		(p.MuxMachineFsm.Machine.Curr.CurrentState() == LacpMuxmStateDetached ||
			p.MuxMachineFsm.Machine.Curr.CurrentState() == LacpMuxmStateCDetached) &&
		p.key == a.actorAdminKey {
		// set port as selected
		p.aggSelected = LacpAggSelected
		// inform mux that port has been selected
		p.MuxMachineFsm.MuxmEvents <- LacpMachineEvent{e: LacpMuxmEventSelectedEqualSelected,
			src:          PortConfigModuleStr,
			responseChan: p.portChan}
		// wait for response
		<-p.portChan
		return true
	}
	return false
}
