// config
package lacp

import (
	"fmt"
	//	"time"
	"sync"
)

const PortConfigModuleStr = "Port Config"

type LaAggConfig struct {
	// Aggregator_MAC_address
	mac [6]uint8
	// Aggregator_Identifier
	Id int
	// Actor_Admin_Aggregator_Key
	Key uint16
	// LAG_ports
	LagMembers []uint16

	// TODO hash config
}

type LaAggPortConfig struct {

	// Actor_Port_Number
	Id uint16
	// Actor_Port_Priority
	Prio uint16
	// Actor Admin Key
	Key uint16
	// Actor Oper Key
	//OperKey uint16
	// Actor_Port_Aggregator_Identifier
	AggId uint16

	// Admin Enable/Disable
	Enable bool

	// lacp mode On/Active/Passive
	Mode int

	// Port capabilities and attributes
	Properties PortProperties

	// Linux If
	IntfId string
}

func CreateLaAgg(agg *LaAggConfig) {

	var p *LaAggPort
	var wg sync.WaitGroup

	a := NewLaAggregator(agg)
	//fmt.Printf("%#v\n", a)

	for _, pId := range a.PortNumList {
		wg.Add(1)
		go func(pId uint16) {
			defer wg.Done()
			if LaFindPortById(pId, &p) && p.aggSelected == LacpAggUnSelected {
				// if aggregation has been provided then lets kick off the process
				p.checkConfigForSelection()
			}
		}(pId)
	}
	wg.Wait()
}

func DeleteLaAgg(Id int) {
	var a *LaAggregator
	if LaFindAggById(Id, &a) {

		for idx, pId := range a.PortNumList {
			DeleteLaAggPort(pId)
			a.PortNumList = append(a.PortNumList[:idx], a.PortNumList[idx+1:]...)
		}

		a.aggId = 0
		a.actorAdminKey = 0
		a.partnerSystemId = [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		a.ready = false
	}
}

func CreateLaAggPort(port *LaAggPortConfig) {
	var pTmp *LaAggPort

	// sanity check that port does not exist already
	if !LaFindPortById(port.Id, &pTmp) {
		p := NewLaAggPort(port)
		//fmt.Printf("%#v\n", p)

		// Is lacp enabled or not
		if port.Mode != LacpModeOn {
			p.lacpEnabled = true
			// make the port aggregatable
			LacpStateSet(&p.actorAdmin.state, LacpStateAggregationBit)
			// set the activity state
			if port.Mode == LacpModePassive {
				LacpStateSet(&p.actorAdmin.state, LacpStateActivityBit)
			} else {
				LacpStateClear(&p.actorAdmin.state, LacpStateActivityBit)
			}
		} else {
			// port is not aggregatible
			LacpStateClear(&p.actorAdmin.state, LacpStateAggregationBit)
			LacpStateSet(&p.actorAdmin.state, LacpStateActivityBit)
			p.lacpEnabled = false
		}

		// lets start all the state machines
		p.BEGIN(false)

		// TODO: need logic to check link status
		p.linkOperStatus = true

		if p.linkOperStatus {

			// if aggregation has been provided then lets kick off the process
			p.checkConfigForSelection()

			if p.aggSelected == LacpAggSelected {
				// if port is enabled and lacp is enabled
				p.LaAggPortEnabled()
			}
		}
	} else {
		fmt.Println("CONF: ERROR PORT ALREADY EXISTS")
	}
}

func DeleteLaAggPort(pId uint16) {
	var p *LaAggPort
	if LaFindPortById(pId, &p) {
		p.DelLaAggPort()
	}
}

func AddLaAggPortToAgg(aggId int, pId uint16) {

	var a *LaAggregator
	var p *LaAggPort

	// both add and port must have existed
	if LaFindAggById(aggId, &a) && LaFindPortById(pId, &p) &&
		p.aggSelected == LacpAggUnSelected {

		// add port to port number list
		a.PortNumList = append(a.PortNumList, p.portNum)
		// add reference to aggId
		p.aggId = aggId
		// well obviously this should pass
		p.checkConfigForSelection()
	}
}
