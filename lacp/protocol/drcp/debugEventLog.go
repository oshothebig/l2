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
	"l2/lacp/protocol/utils"
)

func (dr *DistributedRelay) LaDrLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (p *DRCPIpp) LaIppLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (rxm *RxMachine) DrcpRxmLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (ptxm *PtxMachine) DrcpPtxmLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (psm *PsMachine) DrcpPsmLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (gm *GMachine) DrcpGmLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (am *AMachine) DrcpAmLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (txm *TxMachine) DrcpTxmLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (nism *NetIplShareMachine) DrcpNetIplSharemLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (iam *IAMachine) DrcpIAmLog(msg string) {
	utils.GlobalLogger.Info(msg)
}

func (igm *IGMachine) DrcpIGmLog(msg string) {
	utils.GlobalLogger.Info(msg)
}
