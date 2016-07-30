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

// def.go
package drcp

import (
	"time"
)

const (
	CONVERSATION_ID_TYPE_PORT    = 0
	CONVERSATION_ID_TYPE_GATEWAY = 1

	MAX_IPP_LINKS                    = 3
	MAX_GATEWAY_CONVERSATIONS        = 4096
	GATEWAY_ALGORITHM_RESERVED       = GatewayAlgorithm{0x00, 0x80, 0xC2, 0x00}
	GATEWAY_ALGORITHM_CVID           = GatewayAlgorithm{0x00, 0x80, 0xC2, 0x01}
	GATEWAY_ALGORITHM_SVID           = GatewayAlgorithm{0x00, 0x80, 0xC2, 0x02}
	GATEWAY_ALGORITHM_ISID           = GatewayAlgorithm{0x00, 0x80, 0xC2, 0x03}
	GATEWAY_ALGORITHM_TE_SID         = GatewayAlgorithm{0x00, 0x80, 0xC2, 0x04}
	GATEWAY_ALGORITHM_ECMP_FLOW_HASH = GatewayAlgorithm{0x00, 0x80, 0xC2, 0x05}

	DrniFastPeriodicTime time.Duration = time.Second * 1
	DrniSlowPeriodictime time.Duration = time.Second * 30
	DrniShortTimeoutTime time.Duration = 3 * DrniFastPeriodicTime
	DrniLongTimeoutTime  time.Duration = 3 * DrniSlowPeriodictime
)

type GatewayAlgorithm [4]uint8
type EncapMethod [4]uint8
type Md5Digest [16]uint8
