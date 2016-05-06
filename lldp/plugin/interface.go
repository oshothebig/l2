package plugin

import (
	"l2/lldp/config"
)

type AsicIntf interface {
	GetPortsInfo() []*config.PortInfo
	Start()
}

type ConfigIntf interface {
}
