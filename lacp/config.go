// config
package lacp

import (
	"fmt"
)

const PortConfigModuleStr = "Port Config"

type LaAggConfig struct {
	Mac     [6]uint8
	Id      int
	Prio    int
	Key     int
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
