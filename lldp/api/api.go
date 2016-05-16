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
package api

import (
	"l2/lldp/config"
	"l2/lldp/server"
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

func SendGlobalConfig(ifIndex int32, enable bool) {
	lldpapi.server.GblCfgCh <- &config.Global{ifIndex, enable}
}

func SendPortStateChange(ifIndex int32, state string) {
	lldpapi.server.IfStateCh <- &config.PortState{ifIndex, state}
}

func GetIntfStates(idx int, cnt int) (int, int, []config.IntfState) {
	n, c, result := lldpapi.server.GetIntfStates(idx, cnt)
	return n, c, result
}
