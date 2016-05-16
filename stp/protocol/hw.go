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
// hw.go
package stp

import (
	hwconst "asicd/asicdCommonDefs"
	"asicd/pluginManager/pluginCommon"
	"asicdInt"
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"utils/ipcutils"
)

// to protect against thrift not being thread safe
var asicdmutex *sync.Mutex

type STPClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	STPClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

var asicdclnt AsicdClient

// look up the various other daemons based on c string
func GetClientPort(paramsFile string, c string) int {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		StpLogger("ERROR", fmt.Sprintf("Error in reading configuration file:%s err:%s\n", paramsFile, err))
		return 0
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		StpLogger("ERROR", "Error in Unmarshalling Json")
		return 0
	}

	for _, client := range clientsList {
		if client.Name == c {
			return client.Port
		}
	}
	return 0
}

// connect the the asic d
func ConnectToClients(paramsFile string) {
	port := GetClientPort(paramsFile, "asicd")
	if port != 0 {

		for {
			asicdclnt.Address = "localhost:" + strconv.Itoa(port)
			asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
			//StpLogger("INFO", fmt.Sprintf("found asicd at port %d Transport %#v PrtProtocolFactory %#v\n", port, asicdclnt.Transport, asicdclnt.PtrProtocolFactory))
			if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
				StpLogger("INFO", "connecting to asicd\n")
				asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
				// lets gather all info needed from asicd such as the port
				ConstructPortConfigMap()
				break
			} else {
				StpLogger("WARNING", "Unable to connect to ASICD, retrying in 500ms")
				time.Sleep(time.Millisecond * 500)
			}
		}
	}
}

func ConstructPortConfigMap() {
	currMarker := asicdServices.Int(hwconst.MIN_SYS_PORTS)
	if asicdclnt.ClientHdl != nil {
		StpLogger("INFO", "Calling asicd for port config")
		count := asicdServices.Int(hwconst.MAX_SYS_PORTS)
		for {
			bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(currMarker, count)
			if err != nil {
				StpLogger("ERROR", fmt.Sprintf("GetBulkPortState Error: %s", err))
				return
			}
			StpLogger("INFO", fmt.Sprintf("Length of GetBulkPortState: %d", bulkInfo.Count))

			bulkCfgInfo, err := asicdclnt.ClientHdl.GetBulkPort(currMarker, count)
			if err != nil {
				StpLogger("ERROR", fmt.Sprintf("Error: %s", err))
				return
			}

			StpLogger("INFO", fmt.Sprintf("Length of GetBulkPortConfig: %d", bulkCfgInfo.Count))
			objCount := int(bulkInfo.Count)
			more := bool(bulkInfo.More)
			currMarker = asicdServices.Int(bulkInfo.EndIdx)
			for i := 0; i < objCount; i++ {
				ifindex := bulkInfo.PortStateList[i].IfIndex
				ent := PortConfigMap[ifindex]
				ent.PortNum = bulkInfo.PortStateList[i].PortNum
				ent.IfIndex = ifindex
				ent.Name = bulkInfo.PortStateList[i].Name
				ent.HardwareAddr, _ = net.ParseMAC(bulkCfgInfo.PortList[i].MacAddr)
				PortConfigMap[ifindex] = ent
				StpLogger("INIT", fmt.Sprintf("Found Port %d IfIndex %d Name %s\n", ent.PortNum, ent.IfIndex, ent.Name))
			}
			if more == false {
				return
			}
		}
	}
}

// convert the lacp port names name to asic format string list
func asicDPortBmpFormatGet(distPortList []string) string {
	s := ""
	dLength := len(distPortList)

	for i := 0; i < dLength; i++ {
		num := strings.Split(distPortList[i], "-")[1]
		if i == dLength-1 {
			s += num
		} else {
			s += num + ","
		}
	}
	return s

}

func asicdGetPortLinkStatus(pId int32) bool {

	if asicdclnt.ClientHdl != nil {
		asicdmutex.Lock()
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(asicdServices.Int(hwconst.MIN_SYS_PORTS), asicdServices.Int(hwconst.MAX_SYS_PORTS))
		asicdmutex.Unlock()
		if err == nil && bulkInfo.Count != 0 {
			objCount := int64(bulkInfo.Count)
			for i := int64(0); i < objCount; i++ {
				if bulkInfo.PortStateList[i].IfIndex == pId {
					return bulkInfo.PortStateList[i].OperState == pluginCommon.UpDownState[1]
				}
			}
		}
		StpLogger("INFO", fmt.Sprintf("asicDGetPortLinkSatus: could not get status for port %d, failure in get method\n", pId))
	}
	return true

}

func asicdCreateStgBridge(vlanList []uint16) int32 {

	vl := make([]int32, 0)
	//StpLogger("INFO", fmt.Sprintf("Created Stg Group vlanList[%#v]", vlanList))

	if asicdclnt.ClientHdl != nil {
		for _, v := range vlanList {
			StpLogger("INFO", fmt.Sprintf("vlan in list %d", v))

			if v == DEFAULT_STP_BRIDGE_VLAN {
				StpLogger("INFO", fmt.Sprintf("Default stg vlan"))
			}
			vl = append(vl, int32(v))
		}
		asicdmutex.Lock()
		// default vlan is already created in opennsl
		stgid, err := asicdclnt.ClientHdl.CreateStg(vl)
		asicdmutex.Unlock()
		if err == nil {
			StpLogger("INFO", fmt.Sprintf("Created Stg Group %d with vlans %#v", stgid, vl))
			for _, v := range vl {
				if v != 0 &&
					v != DEFAULT_STP_BRIDGE_VLAN {
					protocolmac := asicdInt.RsvdProtocolMacConfig{
						MacAddr:     "01:00:0C:CC:CC:CD",
						MacAddrMask: "FF:FF:FF:FF:FF:FF",
						VlanId:      int32(v),
					}
					StpLogger("INFO", fmt.Sprintf("Creating PVST MAC entry %#v", protocolmac))
					asicdmutex.Lock()
					asicdclnt.ClientHdl.EnablePacketReception(&protocolmac)
					asicdmutex.Unlock()
				}
			}
			return stgid
		} else {
			StpLogger("INFO", fmt.Sprintf("Create Stg Group error %#v", err))
		}
	} else {
		StpLogger("INFO", fmt.Sprintf("Create Stg Group failed asicd not connected"))
	}
	return -1
}

func asicdDeleteStgBridge(stgid int32, vlanList []uint16) error {
	vl := make([]int32, 0)

	if asicdclnt.ClientHdl != nil {

		for _, v := range vlanList {
			vl = append(vl, int32(v))
		}
		for _, v := range vl {
			if v != 0 &&
				v != DEFAULT_STP_BRIDGE_VLAN {
				protocolmac := asicdInt.RsvdProtocolMacConfig{
					MacAddr:     "01:00:0C:CC:CC:CD",
					MacAddrMask: "FF:FF:FF:FF:FF:FF",
					VlanId:      int32(v),
				}

				StpLogger("INFO", fmt.Sprintf("Deleting PVST MAC entry %#v", protocolmac))
				asicdmutex.Lock()
				asicdclnt.ClientHdl.DisablePacketReception(&protocolmac)
				asicdmutex.Unlock()
			}
		}
		StpLogger("INFO", fmt.Sprintf("Deleting Stg Group %d with vlans %#v", stgid, vl))

		asicdmutex.Lock()
		_, err := asicdclnt.ClientHdl.DeleteStg(stgid)
		asicdmutex.Unlock()
		if err != nil {
			return err
		}
	}
	return nil
}

func asicdSetStgPortState(stgid int32, ifindex int32, state int) error {
	if asicdclnt.ClientHdl != nil {
		for _, pc := range PortConfigMap {
			if pc.IfIndex == ifindex {
				asicdmutex.Lock()
				_, err := asicdclnt.ClientHdl.SetPortStpState(stgid, pc.PortNum, int32(state))
				asicdmutex.Unlock()
				return err
			}
		}
	}
	return nil
}

func asicdFlushFdb(stgid int32) error {
	if asicdclnt.ClientHdl != nil {
		asicdmutex.Lock()
		_, err := asicdclnt.ClientHdl.FlushFdbStgGroup(stgid)
		asicdmutex.Unlock()
		return err
	}
	return nil
}

func asicdBPDUGuardDetected(ifindex int32, enable bool) error {
	if asicdclnt.ClientHdl != nil {
		state := "DOWN"
		if enable {
			state = "UP"
		}
		asicdmutex.Lock()
		_, err := asicdclnt.ClientHdl.ErrorDisablePort(ifindex, state, "STP BPDU GUARD")
		asicdmutex.Unlock()
		return err
	}
	return nil
}
