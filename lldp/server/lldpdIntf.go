package lldpServer

import (
	"asicd/asicdCommonDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l2/lldp/utils"
	_ "utils/commonDefs"
)

/* Register with Asicd and then get l2 port info from asicd via GetBulk
 */
func (svr *LLDPServer) GetInfoFromAsicd() error {
	debug.Logger.Info("Calling Asicd to initialize port properties")
	err := svr.RegisterWithAsicdUpdates(asicdCommonDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go svr.AsicdSubscriber()
	}
	// Get L2 Port States
	svr.GetPortStates()

	// Get L2 Port's
	svr.GetPorts()

	return nil
}

/* Helper function which will connect with asicd, so that any future events from
 * asicd will be handled from lldpServer for lldp frames.
 */
func (svr *LLDPServer) RegisterWithAsicdUpdates(address string) error {
	var err error
	debug.Logger.Info("setting up asicd update listener")
	if svr.asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to create ASIC subscribe",
			"socket, error:", err))
		return err
	}

	if err = svr.asicdSubSocket.Subscribe(""); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on",
			"ASIC subscribe socket, error:",
			err))
		return err
	}

	if _, err = svr.asicdSubSocket.Connect(address); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to connect to ASIC",
			"publisher socket, address:", address, "error:", err))
		return err
	}

	if err = svr.asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to set the buffer size for ",
			"ASIC publisher socket, error:", err))
		return err
	}
	debug.Logger.Info("asicd update listener is set")
	return nil
}

/* Go routine to listen all asicd events notifications.
 * Today lldp listens to only l2 state change. Add other notifications as needed
 */
func (svr *LLDPServer) AsicdSubscriber() {
	for {
		rxBuf, err := svr.asicdSubSocket.Recv(0)
		if err != nil {
			debug.Logger.Err(fmt.Sprintln("Recv on asicd Subscriber",
				"socket failed with error:", err))
			continue
		}
		var msg asicdCommonDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			debug.Logger.Err(fmt.Sprintln("Unable to Unmarshal",
				"asicd msg:", msg.Msg))
			continue
		}
		if msg.MsgType == asicdCommonDefs.NOTIFY_L2INTF_STATE_CHANGE {
			var l2IntfStateNotifyMsg asicdCommonDefs.L2IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l2IntfStateNotifyMsg)
			if err != nil {
				debug.Logger.Err(fmt.Sprintln("Unable to Unmarshal l2 intf",
					"state change:", msg.Msg))
				continue
			}
			svr.UpdateL2IntfStateChange(l2IntfStateNotifyMsg)
		}
	}
}

/*  Helper function to get bulk port state information from asicd
 */
func (svr *LLDPServer) GetPortStates() {
	debug.Logger.Info("Get Port State List")
	currMarker := int64(asicdCommonDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	for {
		bulkInfo, err := svr.asicdClient.ClientHdl.GetBulkPortState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			debug.Logger.Err(fmt.Sprintln(": getting bulk port config"+
				" from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			svr.InitL2PortInfo(bulkInfo.PortStateList[i])
		}
		if more == false {
			break
		}
	}
}

/*  Helper function to get bulk port state information from asicd
 */
func (svr *LLDPServer) GetPorts() {
	debug.Logger.Info("Get Port List")
	currMarker := int64(asicdCommonDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	for {
		bulkInfo, err := svr.asicdClient.ClientHdl.GetBulkPort(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			debug.Logger.Err(fmt.Sprintln(": getting bulk port config"+
				" from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			svr.UpdateL2PortInfo(bulkInfo.PortList[i])
		}
		if more == false {
			break
		}
	}
}

/*  handle l2 state up/down notifications..
 */
func (svr *LLDPServer) UpdateL2IntfStateChange(
	updateInfo asicdCommonDefs.L2IntfStateNotifyMsg) {
	gblInfo, found := svr.lldpGblInfo[updateInfo.IfIndex]
	if !found {
		return
	}
	switch updateInfo.IfState {
	case asicdCommonDefs.INTF_STATE_UP:
		debug.Logger.Info("State UP notification for " + gblInfo.Name)
		gblInfo.OperStateLock.Lock()
		gblInfo.OperState = LLDP_PORT_STATE_UP
		svr.lldpGblInfo[updateInfo.IfIndex] = gblInfo
		gblInfo.OperStateLock.Unlock()
		// Create Pcap Handler and start rx/tx packets
		svr.StartRxTx(updateInfo.IfIndex)
	case asicdCommonDefs.INTF_STATE_DOWN:
		debug.Logger.Info("State DOWN notification for " + gblInfo.Name)
		gblInfo.OperStateLock.Lock()
		gblInfo.OperState = LLDP_PORT_STATE_DOWN
		gblInfo.OperStateLock.Unlock()
		svr.lldpGblInfo[updateInfo.IfIndex] = gblInfo
		// Delete Pcap Handler and stop rx/tx packets
		svr.StopRxTx(updateInfo.IfIndex)
	}
}
