Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
// init.go
package stp

import (
	"sync"
	"utils/logging"
)

var gLogger *logging.Writer

func init() {
	PortConfigMap = make(map[int32]portConfig)
	PortMapTable = make(map[PortMapKey]*StpPort, 0)
	BridgeMapTable = make(map[BridgeKey]*Bridge, 0)
	StpPortConfigMap = make(map[int32]StpPortConfig, 0)
	StpBridgeConfigMap = make(map[int32]StpBridgeConfig, 0)

	asicdmutex = &sync.Mutex{}

	// Init the state string maps
	TimerTypeStrStateMapInit()
	PtmMachineStrStateMapInit()
	PrxmMachineStrStateMapInit()
	PrsMachineStrStateMapInit()
	PrtMachineStrStateMapInit()
	BdmMachineStrStateMapInit()
	PimMachineStrStateMapInit()
	PpmmMachineStrStateMapInit()
	TcMachineStrStateMapInit()
	PtxmMachineStrStateMapInit()
	PstMachineStrStateMapInit()

	// create the logger used by this module
	gLogger, _ = logging.NewLogger("stpd", "STP", true)

}
