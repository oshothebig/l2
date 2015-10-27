// config
package lacp

import (
	"fmt"
)

const PortConfigModuleStr = "Port Config"

type LaAggConfig struct {
	// Aggregator_MAC_address
	mac [6]uint8
	// Aggregator_Identifier
	Id int
	// Actor_Admin_Aggregator_Key
	Key int
	// Actor_Oper_Aggregator_Key
	OperKey int
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
	OperKey uint16
	// Actor_Port_Aggregator_Identifier
	AggId uint16

	// lacp mode On/Active/Passive
	mode int

	// Port capabilities and attributes
	Properties PortProperties

	// Linux If
	IntfId string
}

func CreateLaAgg(agg *LaAggConfig) {

	a := NewLaAggregator(agg)
	fmt.Printf("%#v\n", a)

}

func CreateLaAggPort(port *LaAggPortConfig) {

	p := NewLaAggPort(port)
	fmt.Printf("%#v\n", p)

	// lets start all the state machines
	p.BEGIN(false)

	// if aggregation has been provided then lets kick off the process
	p.checkConfigForSelection()
}

func AddLaAggPortToAgg() {

}
