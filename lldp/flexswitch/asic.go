package flexswitch

import (
	"asicd/asicdCommonDefs"
	"asicdServices"
	"encoding/json"
	"errors"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l2/lldp/api"
	"l2/lldp/config"
	"l2/lldp/utils"
	"strconv"
	"time"
	"utils/ipcutils"
)

type AsicPlugin struct {
	asicdClient    *asicdServices.ASICDServicesClient
	asicdSubSocket *nanomsg.SubSocket
}

func connectAsicd(filePath string, asicdClient chan *asicdServices.ASICDServicesClient) {
	fileName := filePath + CLIENTS_FILE_NAME

	clientJson, err := getClient(fileName, "asicd")
	if err != nil || clientJson == nil {
		asicdClient <- nil
		return
	}

	clientTransport, protocolFactory, err := ipcutils.CreateIPCHandles("localhost:" +
		strconv.Itoa(clientJson.Port))
	if err != nil {
		debug.Logger.Info("Failed to connect to ASICd, retrying until success")
		count := 0
		ticker := time.NewTicker(time.Duration(250) * time.Millisecond)
		for _ = range ticker.C {
			clientTransport, protocolFactory, err =
				ipcutils.CreateIPCHandles("localhost:" +
					strconv.Itoa(clientJson.Port))
			if err == nil {
				ticker.Stop()
				break
			}
			count++
			if (count % 10) == 0 {
				debug.Logger.Info("Still waiting to connect to ASICd")
			}
		}
	}
	client := asicdServices.NewASICDServicesClientFactory(clientTransport,
		protocolFactory)
	asicdClient <- client
}

func NewAsicPlugin(fileName string) (*AsicPlugin, error) {
	var asicdClient *asicdServices.ASICDServicesClient = nil
	asicdClientCh := make(chan *asicdServices.ASICDServicesClient)

	debug.Logger.Info("Connecting to ASICd")
	go connectAsicd(fileName, asicdClientCh)
	asicdClient = <-asicdClientCh
	if asicdClient == nil {
		debug.Logger.Err("Failed to connecto to ASICd")
		return nil, errors.New("Failed to connect to ASICd")
	}

	mgr := &AsicPlugin{
		asicdClient: asicdClient,
	}
	return mgr, nil

}

/*  Helper function to get bulk port state information from asicd
 */
func (p *AsicPlugin) getPortStates() []*config.PortInfo {
	debug.Logger.Info("Get Port State List")
	currMarker := int64(asicdCommonDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	portStates := make([]*config.PortInfo, 0)
	for {
		bulkInfo, err := p.asicdClient.GetBulkPortState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			debug.Logger.Err(fmt.Sprintln(": getting bulk port config"+
				" from asicd failed with reason", err))
			//return
			break
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			obj := bulkInfo.PortStateList[i]
			port := &config.PortInfo{
				IfIndex:   obj.IfIndex,
				PortNum:   obj.PortNum,
				OperState: obj.OperState,
				Name:      obj.Name,
			}
			pObj, err := p.asicdClient.GetPort(obj.IfIndex)
			if err != nil {
				debug.Logger.Err(fmt.Sprintln("Getting mac address for",
					obj.Name, "failed, error:", err))
			} else {
				port.MacAddr = pObj.MacAddr
			}
			portStates = append(portStates, port)
		}
		if more == false {
			break
		}
	}
	debug.Logger.Info("Done with Port State list")
	return portStates
}

/*  Helper function to get bulk port state information from asicd
 */
/*
func (p *AsicPlugin) getPorts(portInfo map[int32]*config.PortInfo,
	portMap map[int32]int32) { //[]string {
	debug.Logger.Info("Get Port List")
	currMarker := int64(asicdCommonDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	//ports := make([]string, 0)
	for {
		bulkInfo, err := p.asicdClient.GetBulkPort(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			debug.Logger.Err(fmt.Sprintln(": getting bulk port config"+
				" from asicd failed with reason", err))
			break
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			ifIndex, exists := portMap[bulkInfo.PortList[i].PortNum]
			if !exists {
				continue
			}
			entry, exists := portInfo[ifIndex]
			if !exists {
				continue
			}
			entry.MacAddr = bulkInfo.PortList[i].MacAddr
			portInfo[ifIndex] = entry
		}
		if more == false {
			break
		}
	}
	debug.Logger.Info("Done with Port list")
	return //ports
}
*/

func (p *AsicPlugin) GetPortsInfo() []*config.PortInfo {
	portStates /*, portMap*/ := p.getPortStates()
	//p.getPorts(portStates, portMap)
	return portStates
}

func (p *AsicPlugin) connectSubSocket() error {
	var err error
	address := asicdCommonDefs.PUB_SOCKET_ADDR
	debug.Logger.Info(" setting up asicd update listener")
	if p.asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to create ASIC subscribe socket, error:",
			err))
		return err
	}

	if err = p.asicdSubSocket.Subscribe(""); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to subscribe to ASIC subscribe socket",
			"error:", err))
		return err
	}

	if _, err = p.asicdSubSocket.Connect(address); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to connect to ASIC publisher socket",
			"address:", address, "error:", err))
		return err
	}

	debug.Logger.Info(fmt.Sprintln(" Connected to ASIC publisher at address:", address))
	if err = p.asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		debug.Logger.Err(fmt.Sprintln(" Failed to set the buffer size for ASIC publisher",
			"socket, error:", err))
		return err
	}
	debug.Logger.Info("asicd update listener is set")
	return nil
}

func (p *AsicPlugin) listenAsicdUpdates() {
	for {
		debug.Logger.Debug(" Read on Asic Subscriber socket....")
		rxBuf, err := p.asicdSubSocket.Recv(0)
		if err != nil {
			debug.Logger.Err(fmt.Sprintln(
				"Recv on asicd Subscriber socket failed with error:", err))
			continue
		}
		var msg asicdCommonDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			debug.Logger.Err(fmt.Sprintln("Unable to Unmarshal asicd msg:", msg.Msg))
			continue
		}
		switch msg.MsgType {
		case asicdCommonDefs.NOTIFY_L2INTF_STATE_CHANGE:
			var l2IntfStateNotifyMsg asicdCommonDefs.L2IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l2IntfStateNotifyMsg)
			if err != nil {
				debug.Logger.Err(fmt.Sprintln("Unable to Unmarshal l2 intf",
					"state change:", msg.Msg))
				continue
			}
			if l2IntfStateNotifyMsg.IfState == asicdCommonDefs.INTF_STATE_UP {
				api.SendPortStateChange(l2IntfStateNotifyMsg.IfIndex, "UP")
			} else {
				api.SendPortStateChange(l2IntfStateNotifyMsg.IfIndex, "DOWN")
			}
			//@TODO: Send API Channel
		}
	}

}
func (p *AsicPlugin) Start() {

	err := p.connectSubSocket()
	if err != nil {
		return
	}
	go p.listenAsicdUpdates()
}
