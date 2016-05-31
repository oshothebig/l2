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

package api

import (
	"errors"
	"l2/lldp/config"
	"l2/lldp/server"
	"strconv"
	"sync"
)

type ApiLayer struct {
	server *server.LLDPServer
}

var lldpapi *ApiLayer = nil
var once sync.Once

/*  Singleton instance should be accesible only within api
 */
func getInstance() *ApiLayer {
	once.Do(func() {
		lldpapi = &ApiLayer{}
	})
	return lldpapi
}

func Init(svr *server.LLDPServer) {
	lldpapi = getInstance()
	lldpapi.server = svr
}

//@TODO: Create for LLDP Interface will only be called during auto-create if an entry is not present in the DB
// If it is present then we will read it during restart
// During update we need to check whether there is any entry in the runtime information or not
func validateExistingIntfConfig(ifIndex int32) (bool, error) {
	exists := lldpapi.server.EntryExist(ifIndex)
	if !exists {
		return exists, errors.New("Update cannot be performed for " +
			strconv.Itoa(int(ifIndex)) + " as LLDP Server doesn't have any info for the ifIndex")
	}
	return exists, nil
}

func SendIntfConfig(ifIndex int32, enable bool) (bool, error) {
	// Validate ifIndex before sending the config to server
	proceed, err := validateExistingIntfConfig(ifIndex)
	if !proceed {
		return proceed, err
	}
	lldpapi.server.IntfCfgCh <- &config.Intf{ifIndex, enable}
	return proceed, err
}

func UpdateIntfConfig(ifIndex int32, enable bool) (bool, error) {
	proceed, err := validateExistingIntfConfig(ifIndex)
	if !proceed {
		return proceed, err
	}
	lldpapi.server.IntfCfgCh <- &config.Intf{ifIndex, enable}
	return proceed, err
}

func SendGlobalConfig(vrf string, enable bool) (bool, error) {
	if lldpapi.server.Global != nil {
		return false, errors.New("Create/Delete on Global Object is not allowed, please do Update")
	}
	lldpapi.server.GblCfgCh <- &config.Global{vrf, enable}
	return true, nil
}

func UpdateGlobalConfig(vrf string, enable bool) (bool, error) {
	if lldpapi.server.Global == nil {
		return false, errors.New("Update can only be performed if the global object for LLDP is created")
	}
	lldpapi.server.GblCfgCh <- &config.Global{vrf, enable}
	return true, nil
}

func SendPortStateChange(ifIndex int32, state string) {
	lldpapi.server.IfStateCh <- &config.PortState{ifIndex, state}
}

func GetIntfStates(idx int, cnt int) (int, int, []config.IntfState) {
	n, c, result := lldpapi.server.GetIntfStates(idx, cnt)
	return n, c, result
}

func UpdateCache() {
	lldpapi.server.UpdateCacheCh <- true
}
