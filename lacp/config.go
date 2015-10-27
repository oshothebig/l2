// config
package lacp

import (
	"fmt"
)

const PortConfigModuleStr = "Port Config"

type LaAggConfig struct {
	Mac     [6]uint8
	Id      uint16
	Prio    uint16
	Key     uint16
	Operkey int
	IntfId  string
}

func CreateLaAgg(agg *LaAggConfig) {

	p := NewLaAggPort(agg)
	fmt.Println(p)

}

func CreateLaAggPort() {

}

func AddLaAggPortToAgg() {

}
