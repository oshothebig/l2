package lldpServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	_ "utils/commonDefs"
)

/* Register with Asicd and then get l2 port info from asicd via GetBulk
 */
func (svr *LLDPServer) LLDPGetInfoFromAsicd() error {
	svr.logger.Info("Calling Asicd to initialize port properties")
	err := svr.LLDPRegisterWithAsicdUpdates(asicdConstDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go svr.LLDPAsicdSubscriber()
	}
	// Get L2 Port List
	svr.LLDPGetPortList()
	return nil
}

/* Helper function which will connect with asicd, so that any future events from
 * asicd will be handled from lldpServer for lldp frames.
 */
func (svr *LLDPServer) LLDPRegisterWithAsicdUpdates(address string) error {
	var err error
	svr.logger.Info("setting up asicd update listener")
	if svr.asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to create ASIC subscribe",
			"socket, error:", err))
		return err
	}

	if err = svr.asicdSubSocket.Subscribe(""); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on",
			"ASIC subscribe socket, error:",
			err))
		return err
	}

	if _, err = svr.asicdSubSocket.Connect(address); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to connect to ASIC",
			"publisher socket, address:", address, "error:", err))
		return err
	}

	if err = svr.asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to set the buffer size for ",
			"ASIC publisher socket, error:", err))
		return err
	}
	svr.logger.Info("asicd update listener is set")
	return nil
}

/* Go routine to listen all asicd events notifications.
 * Today lldp listens to only l2 state change. Add other notifications as needed
 */
func (svr *LLDPServer) LLDPAsicdSubscriber() {
	for {
		rxBuf, err := svr.asicdSubSocket.Recv(0)
		if err != nil {
			svr.logger.Err(fmt.Sprintln("Recv on asicd Subscriber",
				"socket failed with error:", err))
			continue
		}
		var msg asicdConstDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			svr.logger.Err(fmt.Sprintln("Unable to Unmarshal",
				"asicd msg:", msg.Msg))
			continue
		}
		if msg.MsgType == asicdConstDefs.NOTIFY_L2INTF_STATE_CHANGE {
			var l2IntfStateNotifyMsg asicdConstDefs.L2IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l2IntfStateNotifyMsg)
			if err != nil {
				svr.logger.Err(fmt.Sprintln("Unable to Unmarshal l2 intf",
					"state change:", msg.Msg))
				continue
			}
			svr.LLDPUpdateL2IntfStateChange(l2IntfStateNotifyMsg)
		}
	}
}

/*  Helper function to get bulk port state information from asicd
 */
func (svr *LLDPServer) LLDPGetPortList() {
	svr.logger.Info("Get Port List")
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	for {
		bulkInfo, err := svr.asicdClient.ClientHdl.GetBulkPortState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			svr.logger.Err(fmt.Sprintln("LLDP: getting bulk port config"+
				" from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			svr.LLDPInitL2PortInfo(bulkInfo.PortStateList[i])
		}
		if more == false {
			break
		}
	}
}

/*  handle l2 state up/down notifications..
 */
func (svr *LLDPServer) LLDPUpdateL2IntfStateChange(
	updateInfo asicdConstDefs.L2IntfStateNotifyMsg) {
	gblInfo, found := svr.lldpGblInfo[updateInfo.IfIndex]
	if !found {
		return
	}
	switch updateInfo.IfState {
	case asicdConstDefs.INTF_STATE_UP:
		gblInfo.OperStateLock.Lock()
		gblInfo.OperState = LLDP_PORT_STATE_UP
		svr.lldpGblInfo[updateInfo.IfIndex] = gblInfo
		gblInfo.OperStateLock.Unlock()
		// Create Pcap Handler and start rx/tx packets
		svr.LLDPCreatePcapHandler(updateInfo.IfIndex)
	case asicdConstDefs.INTF_STATE_DOWN:
		gblInfo.OperStateLock.Lock()
		gblInfo.OperState = LLDP_PORT_STATE_DOWN
		svr.lldpGblInfo[updateInfo.IfIndex] = gblInfo
		gblInfo.OperStateLock.Unlock()
		// Delete Pcap Handler and stop rx/tx packets
		svr.LLDPDeletePcapHandler(updateInfo.IfIndex)
	}
}
