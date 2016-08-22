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

// hw.go
package lacp

import (
	hwconst "asicd/asicdCommonDefs"
	"strings"
)

// convert the lacp port names name to asic format string list
func asicDPortBmpFormatGet(distPortList []string) string {
	s := ""
	dLength := len(distPortList)

	for i := 0; i < dLength; i++ {
		var num string
		if strings.Contains(distPortList[i], "-") {
			num = strings.Split(distPortList[i], "-")[1]
		} else if strings.HasPrefix(distPortList[i], "eth") {
			num = strings.TrimLeft(distPortList[i], "eth")
		} else if strings.HasPrefix(distPortList[i], "fpPort") {
			num = strings.TrimLeft(distPortList[i], "fpPort")
		}
		if i == dLength-1 {
			s += num
		} else {
			s += num + ","
		}
	}
	return s

}

// convert the model value to asic value
func asicDHashModeGet(hashmode uint32) (laghash int32) {
	switch hashmode {
	case 0: //L2
		laghash = hwconst.HASH_SEL_SRCDSTMAC
		break
	case 1: //L2 + L3
		laghash = hwconst.HASH_SEL_SRCDSTIP
		break
	//case 2: //L3 + L4
	//break
	default:
		laghash = hwconst.HASH_SEL_SRCDSTMAC
	}
	return laghash
}
