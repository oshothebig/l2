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

package server

import (
	"fmt"
	"l2/lldp/config"
	"l2/lldp/plugin"
	"l2/lldp/utils"
	_ "models/objects"
	"os"
	"os/signal"
	_ "runtime/pprof"
	"strconv"
	"syscall"
	"time"
	"utils/dbutils"
)

/* Create lldp server object for the main handler..
 */
func LLDPNewServer(aPlugin plugin.AsicIntf, lPlugin plugin.ConfigIntf, sPlugin plugin.SystemIntf,
	dbHdl *dbutils.DBUtil) *LLDPServer {
	lldpServerInfo := &LLDPServer{
		asicPlugin: aPlugin,
		CfgPlugin:  lPlugin,
		SysPlugin:  sPlugin,
		lldpDbHdl:  dbHdl,
	}
	// Allocate memory to all the Data Structures
	lldpServerInfo.InitGlobalDS()
	/*
		// Profiling code for lldp
		prof, err := os.Create(LLDP_CPU_PROFILE_FILE)
		if err == nil {
			pprof.StartCPUProfile(prof)
		}
	*/
	return lldpServerInfo
}

/* Allocate memory to all the object which are being used by LLDP server
 */
func (svr *LLDPServer) InitGlobalDS() {
	svr.lldpGblInfo = make(map[int32]LLDPGlobalInfo, LLDP_INITIAL_GLOBAL_INFO_CAPACITY)
	svr.lldpRxPktCh = make(chan InPktChannel, LLDP_RX_PKT_CHANNEL_SIZE)
	svr.lldpTxPktCh = make(chan SendPktChannel, LLDP_TX_PKT_CHANNEL_SIZE)
	svr.lldpExit = make(chan bool)
	svr.lldpSnapshotLen = 1024
	svr.lldpPromiscuous = false
	// LLDP Notifications are atleast 5 seconds apart with default being
	// 30 seconds. So, we can have the leavrage the pcap timeout (read from
	// buffer) to be 1 second.
	svr.lldpTimeout = 1 * time.Second
	svr.GblCfgCh = make(chan *config.Global)
	svr.IntfCfgCh = make(chan *config.Intf)
	svr.IfStateCh = make(chan *config.PortState)
	svr.UpdateCacheCh = make(chan bool)
	svr.EventCh = make(chan config.EventInfo, 10)
	// All Plugin Info
}

/* De-Allocate memory to all the object which are being used by LLDP server
 */
func (svr *LLDPServer) DeInitGlobalDS() {
	svr.lldpRxPktCh = nil
	svr.lldpTxPktCh = nil
	svr.lldpGblInfo = nil
}

/* On de-init we will be closing all the pcap handlers that are opened up
 * We will also free up all the pointers from the gblInfo. Otherwise that will
 * lead to memory leak
 */
func (svr *LLDPServer) CloseAllPktHandlers() {
	// close rx packet channel
	close(svr.lldpRxPktCh)
	close(svr.lldpTxPktCh)

	// close pcap, stop cache timer and free any allocated memory
	for i := 0; i < len(svr.lldpIntfStateSlice); i++ {
		key := svr.lldpIntfStateSlice[i]
		gblInfo, exists := svr.lldpGblInfo[key]
		if !exists {
			continue
		}
		gblInfo.DeInitRuntimeInfo()
		svr.lldpGblInfo[key] = gblInfo
	}
	debug.Logger.Info("closed everything")
}

/* Create global run time information for l2 port and then start rx/tx for that port if state is up
 */
func (svr *LLDPServer) InitL2PortInfo(portInfo *config.PortInfo) {
	gblInfo, exists := svr.lldpGblInfo[portInfo.IfIndex]
	gblInfo.InitRuntimeInfo(portInfo)
	if !exists {
		// on fresh start it will not exists but on restart it might
		// default is set to true but LLDP Object is auto-discover and hence we will enable it manually
		gblInfo.Enable()
	}
	svr.lldpGblInfo[portInfo.IfIndex] = gblInfo

	svr.lldpIntfStateSlice = append(svr.lldpIntfStateSlice, gblInfo.Port.IfIndex)
}

/*  lldp server: 1) Connect to all the clients
 *		 2) Initialize DB
 *		 3) Read from DB and close DB
 *		 4) Call AsicPlugin for port information
 *		 5) go routine to handle all the channels within lldp server
 */
func (svr *LLDPServer) LLDPStartServer(paramsDir string) {
	// OS Signal channel listener thread
	svr.OSSignalHandle()

	svr.paramsDir = paramsDir
	// Initialize DB
	err := svr.InitDB()
	if err != nil {
		debug.Logger.Err("DB init failed")
	} else {
		// Populate Gbl Configs
		svr.ReadDB()
	}
	// Get Port Information from Asic, only after reading from DB
	portsInfo := svr.asicPlugin.GetPortsInfo()
	for _, port := range portsInfo {
		svr.InitL2PortInfo(port)
	}

	svr.asicPlugin.Start()
	svr.SysPlugin.Start()
	go svr.ChannelHanlder()
}

/*  Create os signal handler channel and initiate go routine for that
 */
func (svr *LLDPServer) OSSignalHandle() {
	sigChannel := make(chan os.Signal, 1)
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChannel, signalList...)
	go svr.SignalHandler(sigChannel)
}

/* OS signal handler.
 *      If the process get a sighup signal then close all the pcap handlers.
 *      After that delete all the memory which was used during init process
 */
func (svr *LLDPServer) SignalHandler(sigChannel <-chan os.Signal) {
	signal := <-sigChannel
	switch signal {
	case syscall.SIGHUP:
		svr.lldpExit <- true
		debug.Logger.Alert("Received SIGHUP Signal")
		svr.CloseAllPktHandlers()
		svr.DeInitGlobalDS()
		svr.CloseDB()
		//pprof.StopCPUProfile()
		debug.Logger.Alert("Exiting!!!!!")
		os.Exit(0)
	default:
		debug.Logger.Info(fmt.Sprintln("Unhandled Signal:", signal))
	}
}

/* Create l2 port pcap handler and then start rx and tx on that pcap
 *	Filter is LLDP_BPF_FILTER = "ether proto 0x88cc"
 * Note: API should only and only do
 *  1) pcap create
 *  2) start go routine for Rx/Tx Frames Packet Handler
 *  3) Add the port to UP List
 */
func (svr *LLDPServer) StartRxTx(ifIndex int32) {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		debug.Logger.Err(fmt.Sprintln("No entry for ifindex", ifIndex))
		return
	}
	// if the port is disabled or lldp globally is disabled then no need to start rx/tx...
	if svr.Global.Enable == false {
		return
	}
	if gblInfo.PcapHandle != nil {
		debug.Logger.Info("Pcap already exist means the port changed it states")
		// Move the port to up state and continue
		svr.lldpUpIntfStateSlice = append(svr.lldpUpIntfStateSlice, gblInfo.Port.IfIndex)
		return // returning because the go routine is already up and running for the port
	}
	err := gblInfo.CreatePcapHandler(svr.lldpSnapshotLen, svr.lldpPromiscuous, svr.lldpTimeout)
	if err != nil {
		debug.Logger.Alert("Creating Pcap Handler for " + gblInfo.Port.Name +
			" failed and hence we will not start LLDP on the port")
		return
	}
	svr.lldpGblInfo[ifIndex] = gblInfo
	debug.Logger.Info("Start lldp frames rx/tx for port:" + gblInfo.Port.Name + " ifIndex:" +
		strconv.Itoa(int(gblInfo.Port.IfIndex)))

	// Everything set up, so now lets start with receiving frames and transmitting frames go routine...
	go svr.ReceiveFrames(gblInfo.PcapHandle, ifIndex)
	svr.TransmitFrames(ifIndex)
	svr.lldpUpIntfStateSlice = append(svr.lldpUpIntfStateSlice, gblInfo.Port.IfIndex)
}

/*  Send Signal for stopping rx/tx go routine and timers as the pcap handler for
 *  the port is deleted
 */
func (svr *LLDPServer) StopRxTx(ifIndex int32) {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		debug.Logger.Err(fmt.Sprintln("No entry for ifIndex", ifIndex))
		return
	}
	// We will stop go routine only when config state is disabled on the port
	//if gblInfo.isEnabled() { //&& gblInfo.Port.OperState == LLDP_PORT_STATE_DOWN {
	//	return
	//}
	// Send go routine kill signal right away before even we do anything else
	gblInfo.RxKill <- true
	debug.Logger.Info("Stop lldp frames rx/tx for port:" + gblInfo.Port.Name +
		" ifIndex:" + strconv.Itoa(int(gblInfo.Port.IfIndex)))

	// stop the timer
	gblInfo.TxInfo.StopTxTimer()
	// Delete Pcap Handler
	gblInfo.DeletePcapHandler()
	// invalid the cache information
	gblInfo.TxInfo.DeleteCacheFrame()
	//gblInfo.killerWaitGroup.Add(2)
	svr.lldpGblInfo[ifIndex] = gblInfo
	svr.DeletePortFromUpState(ifIndex)
}

/*  helper function to inform whether rx channel is closed or open...
 *  Go routine can be exited using this information
 */
func (svr *LLDPServer) ServerRxChClose() bool {
	if svr.lldpRxPktCh == nil {
		return true
	}
	return false
}

/*  delete ifindex from lldpUpIntfStateSlice on port down... we can use this
 *  if user decides to disable lldp on a port
 */
func (svr *LLDPServer) DeletePortFromUpState(ifIndex int32) {
	for idx, _ := range svr.lldpUpIntfStateSlice {
		if svr.lldpUpIntfStateSlice[idx] == ifIndex {
			svr.lldpUpIntfStateSlice = append(svr.lldpUpIntfStateSlice[:idx],
				svr.lldpUpIntfStateSlice[idx+1:]...)
			break
		}
	}
}

/*  handle l2 state up/down notifications..
 */
func (svr *LLDPServer) UpdateL2IntfStateChange(ifIndex int32, state string) {
	gblInfo, found := svr.lldpGblInfo[ifIndex]
	if !found {
		return
	}
	switch state {
	case "UP":
		debug.Logger.Debug("State UP notification for " + gblInfo.Port.Name + " ifIndex: " +
			strconv.Itoa(int(gblInfo.Port.IfIndex)))
		gblInfo.Port.OperState = LLDP_PORT_STATE_UP
		svr.lldpGblInfo[ifIndex] = gblInfo
		if gblInfo.isEnabled() {
			// Create Pcap Handler and start rx/tx packets
			svr.StartRxTx(ifIndex)
		}
	case "DOWN":
		debug.Logger.Debug("State DOWN notification for " + gblInfo.Port.Name + " ifIndex: " +
			strconv.Itoa(int(gblInfo.Port.IfIndex)))
		gblInfo.Port.OperState = LLDP_PORT_STATE_DOWN
		svr.lldpGblInfo[ifIndex] = gblInfo
		if gblInfo.isEnabled() {
			// Delete Pcap Handler and stop rx/tx packets
			svr.StopRxTx(ifIndex)
		}
	}
}

/*  handle global lldp enable/disable, which will enable/disable lldp for all the ports
 */
func (svr *LLDPServer) handleGlobalConfig(restart bool) {
	// iterate over all the entries in the gblInfo and change the state accordingly
	for _, ifIndex := range svr.lldpIntfStateSlice {
		gblInfo, found := svr.lldpGblInfo[ifIndex]
		if !found {
			debug.Logger.Err("No entry for ifIndex", ifIndex, "in runtime information")
			continue
		}
		if gblInfo.isDisabled() || gblInfo.Port.OperState == LLDP_PORT_STATE_DOWN {
			debug.Logger.Debug("Cannot start LLDP rx/tx for port", gblInfo.Port.Name,
				"as its state is", gblInfo.Port.OperState, "enable is", gblInfo.isDisabled())
			continue
		}
		switch svr.Global.Enable {
		case true:
			debug.Logger.Debug("Global Config Disable, enabling port rx tx for ifIndex", ifIndex)
			svr.StartRxTx(ifIndex)
		case false:
			if restart {
				continue
			}
			debug.Logger.Debug("Global Config Disable, disabling port rx tx for ifIndex", ifIndex)
			// do not update the configuration enable/disable state...just stop packet handling
			svr.StopRxTx(ifIndex)
		}
	}
}

/*  handle configuration coming from user, which will enable/disable lldp per port
 */
func (svr *LLDPServer) handleIntfConfig(ifIndex int32, enable bool) {
	gblInfo, found := svr.lldpGblInfo[ifIndex]
	if !found {
		debug.Logger.Err(fmt.Sprintln("No entry for ifIndex", ifIndex, "in runtime information"))
		return
	}
	switch enable {
	case true:
		debug.Logger.Debug("Config Enable for " + gblInfo.Port.Name + " ifIndex: " +
			strconv.Itoa(int(gblInfo.Port.IfIndex)))
		gblInfo.Enable()
		svr.lldpGblInfo[ifIndex] = gblInfo
		svr.StartRxTx(ifIndex)
	case false:
		debug.Logger.Debug("Config Disable for " + gblInfo.Port.Name + " ifIndex: " +
			strconv.Itoa(int(gblInfo.Port.IfIndex)))
		if gblInfo.isEnabled() { // If Enabled then only do stop rx/tx
			gblInfo.Disable()
			svr.lldpGblInfo[ifIndex] = gblInfo
			svr.StopRxTx(ifIndex)
		}
	}
}

/*  API to send a frame when tx timer expires per port
 */
func (svr *LLDPServer) SendFrame(ifIndex int32) {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	// extra check for pcap handle
	if exists && gblInfo.PcapHandle != nil {
		if gblInfo.TxInfo.UseCache() == false {
			svr.GetSystemInfo()
		}
		rv := gblInfo.WritePacket(gblInfo.TxInfo.SendFrame(gblInfo.Port, svr.SysInfo))
		if rv == false {
			gblInfo.TxInfo.SetCache(rv)
		}
		svr.lldpGblInfo[ifIndex] = gblInfo
	}
	gblInfo.TxDone <- true
}

/* To handle all the channels in lldp server... For detail look at the
 * LLDPInitGlobalDS api to see which all channels are getting initialized
 */
func (svr *LLDPServer) ChannelHanlder() {
	// Only start rx/tx if, Globally LLDP is enabled, Interface LLDP is enabled and port is in UP state
	// move RX/TX to Channel Handler
	// The below step is really important for us.
	// On Re-Start if lldp global is enable then we will start rx/tx for those ports which are in up state
	// and at the same time we will start the loop for signal handler
	// @TODO: should this be moved to timer... like wait 1 second and then start the handleGlobalConfig??
	svr.handleGlobalConfig(true)

	for {
		select {
		case rcvdInfo, ok := <-svr.lldpRxPktCh:
			if !ok {
				debug.Logger.Alert("RX Channel is closed, exit")
				return // rx channel should be closed only on exit
			}
			gblInfo, exists := svr.lldpGblInfo[rcvdInfo.ifIndex]
			if exists {
				var err error
				eventInfo := config.EventInfo{}
				gblInfo.RxLock.Lock()
				eventInfo.EventType, err = gblInfo.RxInfo.Process(gblInfo.RxInfo, rcvdInfo.pkt)
				if err != nil {
					gblInfo.RxLock.Unlock()
					debug.Logger.Err(fmt.Sprintln("err", err, "while processing rx frame on port",
						gblInfo.Port.Name))
					continue
				}
				gblInfo.RxLock.Unlock()
				// reset/start timer for recipient information
				gblInfo.RxInfo.CheckPeerEntry(gblInfo.Port.Name, svr.EventCh, rcvdInfo.ifIndex)
				svr.lldpGblInfo[rcvdInfo.ifIndex] = gblInfo
				//eventInfo.Info = svr.GetIntfState(rcvdInfo.ifIndex)
				eventInfo.IfIndex = rcvdInfo.ifIndex

				if eventInfo.EventType != config.NoOp {
					svr.SysPlugin.PublishEvent(eventInfo)
				}
				// dump the frame
				//gblInfo.DumpFrame()
			}
		case exit := <-svr.lldpExit:
			if exit {
				debug.Logger.Alert("lldp exiting stopping all" +
					" channel handlers")
				return
			}
		case info, ok := <-svr.lldpTxPktCh:
			if !ok {
				debug.Logger.Alert("TX Channel is closed, exit")
				return
			}
			svr.SendFrame(info.ifIndex)
		case gbl, ok := <-svr.GblCfgCh: // Change in global config
			if !ok {
				continue
			}
			debug.Logger.Info(fmt.Sprintln("Server Received Global Config", gbl))
			if svr.Global == nil {
				svr.Global = &config.Global{}
			}
			svr.Global.Enable = gbl.Enable
			svr.Global.Vrf = gbl.Vrf
			svr.handleGlobalConfig(false)
		case intf, ok := <-svr.IntfCfgCh: // Change in interface config
			if !ok {
				continue
			}
			debug.Logger.Info(fmt.Sprintln("Server received Intf Config", intf))
			svr.handleIntfConfig(intf.IfIndex, intf.Enable)
		case ifState, ok := <-svr.IfStateCh: // Change in Port State..
			if !ok {
				continue
			}
			svr.UpdateL2IntfStateChange(ifState.IfIndex, ifState.IfState)
		case _, ok := <-svr.UpdateCacheCh:
			if !ok {
				continue
			}
			svr.UpdateCache()
		case eventInfo, ok := <-svr.EventCh: //used only for delete
			if !ok {
				continue
			}
			//eventInfo.Info = svr.GetIntfState(eventInfo.IfIndex)
			svr.SysPlugin.PublishEvent(eventInfo)
		}
	}
}
