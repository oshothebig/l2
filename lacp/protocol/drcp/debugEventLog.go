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

// debugEventLog this code is meant to serialize the logging States
package drcp

import (
	//"fmt"
	"strings"
	"utils/logging"
)

var gLogger *logging.Writer

type DrcpDebug struct {
	LacpLogChan chan string
	logger      *logging.Writer
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewDrcpDebug() *DrcpDebug {
	if gLogger == nil {
		gLogger, _ = logging.NewLogger("lacpd", "DRCP", true)
	}
	drcpdebug := &DrcpDebug{
		LogChan: make(chan string, 100),
		logger:  gLogger,
	}

	return drcpdebug
}

func GetLacpLogger() *logging.Writer {
	return gLogger
}

func (l *LacpDebug) Stop() {
	close(l.LogChan)
	if gLogger != nil {
		gLogger.Close()
		gLogger = nil
	}
}

func (p *DRCPIpp) DrcpDebugEventLogMain() {

	p.DrcpDebug = NewLacpDebug()

	go func(port *DRCPIpp) {

		for {
			select {

			case msg, logEvent := <-port.DrcpDebug.LogChan:
				if logEvent {
					port.DrcpDebug.logger.Info(strings.Join([]string{p.IntfNum, msg}, "-"))
				} else {
					return
				}
			}
		}
	}(p)
}

func (rxm *RxMachine) DrcpRxmLog(msg string) {
	if rxm.Machine.Curr.IsLoggerEna() {
		rxm.log <- strings.Join([]string{"DRXM", msg}, ":")
	}
}
