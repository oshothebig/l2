package server

import (
	"fmt"
	"l2/lldp/config"
	"l2/lldp/plugin"
	"l2/lldp/utils"
	"os"
	"os/signal"
	_ "runtime/pprof"
	"syscall"
	"time"
)

/* Create lldp server object for the main handler..
 */
func LLDPNewServer(aPlugin plugin.AsicIntf, lPlugin plugin.ConfigIntf) *LLDPServer {
	lldpServerInfo := &LLDPServer{
		asicPlugin: aPlugin,
		CfgPlugin:  lPlugin,
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
	svr.IfStateCh = make(chan *config.PortState)

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
	gblInfo, _ := svr.lldpGblInfo[portInfo.IfIndex]
	gblInfo.InitRuntimeInfo(portInfo)
	svr.lldpGblInfo[portInfo.IfIndex] = gblInfo

	// Only start rx/tx once we have got the mac address from the get bulk port
	gblInfo.OperStateLock.RLock()
	if gblInfo.Port.OperState == LLDP_PORT_STATE_UP {
		gblInfo.OperStateLock.RUnlock()
		svr.StartRxTx(gblInfo.Port.IfIndex)
	} else {
		gblInfo.OperStateLock.RUnlock()
	}
}

/*  lldp server: 1) Connect to all the clients
 *		 2) Initialize DB
 *		 3) Read from DB and close DB
 *		 5) go routine to handle all the channels within lldp server
 */
func (svr *LLDPServer) LLDPStartServer(paramsDir string) {
	// OS Signal channel listener thread
	svr.OSSignalHandle()

	svr.paramsDir = paramsDir
	// Start Api Layer
	//api.Init(svr.GblCfgCh, svr.IfStateCh)
	// Get Port Information from Asic
	portsInfo := svr.asicPlugin.GetPortsInfo()
	for _, port := range portsInfo {
		svr.InitL2PortInfo(port) // is it a bug for starting rx/tx before channel handler??
	}

	// Initialize DB
	err := svr.InitDB()
	if err != nil {
		debug.Logger.Err("DB init failed")
	} else {
		// Populate Gbl Configs
		svr.ReadDB()
		svr.CloseDB()
	}
	svr.asicPlugin.Start()
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
		//pprof.StopCPUProfile()
		debug.Logger.Alert("Exiting!!!!!")
		os.Exit(0)
	default:
		debug.Logger.Info(fmt.Sprintln("Unhandled Signal:", signal))
	}
}

/* To handle all the channels in lldp server... For detail look at the
 * LLDPInitGlobalDS api to see which all channels are getting initialized
 */
func (svr *LLDPServer) ChannelHanlder() {
	for {
		select {
		case rcvdInfo, ok := <-svr.lldpRxPktCh:
			if !ok {
				debug.Logger.Alert("RX Channel is closed, exit")
				return // rx channel should be closed only on exit
			}
			gblInfo, exists := svr.lldpGblInfo[rcvdInfo.ifIndex]
			if exists {
				err := gblInfo.RxInfo.Process(gblInfo.RxInfo, rcvdInfo.pkt)
				if err != nil {
					debug.Logger.Err(fmt.Sprintln("err", err,
						" while processing rx frame on port",
						gblInfo.Port.Name))
					continue
				}
				// reset/start timer for recipient information
				gblInfo.RxInfo.CheckPeerEntry(gblInfo.Port.Name)
				svr.lldpGblInfo[rcvdInfo.ifIndex] = gblInfo
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
			gblInfo, exists := svr.lldpGblInfo[info.ifIndex]
			// extra check for pcap handle
			if exists && gblInfo.PcapHandle != nil {
				rv := gblInfo.WritePacket(
					gblInfo.TxInfo.SendFrame(gblInfo.Port.MacAddr, gblInfo.Port.Name))
				if rv == false {
					gblInfo.TxInfo.SetCache(rv)
				}
				svr.lldpGblInfo[info.ifIndex] = gblInfo
			}
		case gbl, ok := <-svr.GblCfgCh:
			if !ok {
				continue
			}
			debug.Logger.Info(fmt.Sprintln("Received Global Config", gbl))
		case ifState, ok := <-svr.IfStateCh:
			if !ok {
				continue
			}
			svr.UpdateL2IntfStateChange(ifState.IfIndex, ifState.IfState)
		}
	}
}

/* Create l2 port pcap handler and then start rx and tx on that pcap
 *	Filter is LLDP_BPF_FILTER = "ether proto 0x88cc"
 */
func (svr *LLDPServer) StartRxTx(ifIndex int32) {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		debug.Logger.Err(fmt.Sprintln("No entry for ifindex", ifIndex))
		return
	}
	err := gblInfo.CreatePcapHandler(svr.lldpSnapshotLen, svr.lldpPromiscuous,
		svr.lldpTimeout)
	if err != nil {
		debug.Logger.Alert("Creating Pcap Handler for " + gblInfo.Port.Name +
			" failed and hence we will not start LLDP on the port")
		return
	}

	svr.lldpGblInfo[ifIndex] = gblInfo
	debug.Logger.Info("Start lldp frames rx/tx for port:" + gblInfo.Port.Name)
	go svr.ReceiveFrames(gblInfo.PcapHandle, ifIndex)
	go svr.TransmitFrames(gblInfo.PcapHandle, ifIndex)
	svr.lldpUpIntfStateSlice = append(svr.lldpUpIntfStateSlice,
		gblInfo.Port.IfIndex)
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
	// stop the timer
	gblInfo.TxInfo.StopTxTimer()
	// delete pcap handler
	gblInfo.DeletePcapHandler()
	// invalid the cache information
	gblInfo.TxInfo.DeleteCacheFrame()
	debug.Logger.Info("Stop lldp frames rx/tx for port:" + gblInfo.Port.Name)
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
		debug.Logger.Info("State UP notification for " + gblInfo.Port.Name)
		gblInfo.OperStateLock.Lock()
		gblInfo.Port.OperState = LLDP_PORT_STATE_UP
		svr.lldpGblInfo[ifIndex] = gblInfo
		gblInfo.OperStateLock.Unlock()
		// Create Pcap Handler and start rx/tx packets
		svr.StartRxTx(ifIndex)
	case "DOWN":
		debug.Logger.Info("State DOWN notification for " + gblInfo.Port.Name)
		gblInfo.OperStateLock.Lock()
		gblInfo.Port.OperState = LLDP_PORT_STATE_DOWN
		gblInfo.OperStateLock.Unlock()
		svr.lldpGblInfo[ifIndex] = gblInfo
		// Delete Pcap Handler and stop rx/tx packets
		svr.StopRxTx(ifIndex)
	}
}
