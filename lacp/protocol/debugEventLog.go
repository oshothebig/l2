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
package lacp

import (
	//"fmt"
	"strings"
	"time"
	"utils/logging"
)

type LacpDebug struct {
	LacpLogChan chan string
	logger      *logging.Writer
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpDebug() *LacpDebug {
	logger, _ := logging.NewLogger("lacpd", "LACP", true)
	lacpdebug := &LacpDebug{
		LacpLogChan: make(chan string, 100),
		logger:      logger,
	}

	return lacpdebug
}

func (l *LacpDebug) Stop() {
	close(l.LacpLogChan)
}

func (p *LaAggPort) LacpDebugEventLogMain() {

	p.LacpDebug = NewLacpDebug()

	go func(port *LaAggPort) {

		for {
			select {

			case msg, logEvent := <-port.LacpDebug.LacpLogChan:
				if logEvent {
					port.LacpDebug.logger.Info(strings.Join([]string{p.IntfNum, msg}, "-"))
				} else {
					return
				}
			}
		}
	}(p)
}

func (a *LaAggregator) LacpDebugAggEventLogMain() {

	a.LacpDebug = NewLacpDebug()

	go func(a *LaAggregator) {

		for {
			select {

			case msg, logEvent := <-a.LacpDebug.LacpLogChan:
				if logEvent {
					a.LacpDebug.logger.Info(strings.Join([]string{a.AggName, msg}, "-"))
				} else {
					return
				}
			}
		}
	}(a)
}

func (a *LaAggregator) LacpAggLog(msg string) {
	a.log <- strings.Join([]string{"AGG", time.Now().String(), msg}, ":")
}

func (txm *LacpTxMachine) LacpTxmLog(msg string) {
	if txm.Machine.Curr.IsLoggerEna() {
		txm.log <- strings.Join([]string{"TXM", time.Now().String(), msg}, ":")
	}
}

func (cdm *LacpCdMachine) LacpCdmLog(msg string) {
	if cdm.Machine.Curr.IsLoggerEna() {
		cdm.log <- strings.Join([]string{"CDM", time.Now().String(), msg}, ":")
	}
}

func (cdm *LacpPartnerCdMachine) LacpCdmLog(msg string) {
	if cdm.Machine.Curr.IsLoggerEna() {
		cdm.log <- strings.Join([]string{"PCDM", time.Now().String(), msg}, ":")
	}
}

func (ptxm *LacpPtxMachine) LacpPtxmLog(msg string) {
	if ptxm.Machine.Curr.IsLoggerEna() {
		ptxm.log <- strings.Join([]string{"PTXM", time.Now().String(), msg}, ":")
	}
}

func (rxm *LacpRxMachine) LacpRxmLog(msg string) {
	if rxm.Machine.Curr.IsLoggerEna() {
		rxm.log <- strings.Join([]string{"RXM", time.Now().String(), msg}, ":")
	}
}

func (muxm *LacpMuxMachine) LacpMuxmLog(msg string) {
	if muxm.Machine.Curr.IsLoggerEna() {
		muxm.log <- strings.Join([]string{"MUXM", time.Now().String(), msg}, ":")
	}
}

func (mr *LampMarkerResponderMachine) LampMarkerResponderLog(msg string) {
	if mr.Machine.Curr.IsLoggerEna() {
		mr.log <- strings.Join([]string{"MARKER RESPONDER", time.Now().String(), msg}, ":")
	}
}
