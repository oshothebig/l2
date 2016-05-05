package server

import (
	"asicdServices"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"l2/lldp/utils"
	"os"
	"os/signal"
	_ "runtime/pprof"
	"strconv"
	_ "sync"
	"syscall"
	"time"
	"utils/ipcutils"
)

/* Create lldp server object for the main handler..
 */
func LLDPNewServer() *LLDPServer {
	lldpServerInfo := &LLDPServer{}
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
	svr.lldpPortNumIfIndexMap = make(map[int32]int32, LLDP_INITIAL_GLOBAL_INFO_CAPACITY)
	svr.lldpRxPktCh = make(chan InPktChannel, LLDP_RX_PKT_CHANNEL_SIZE)
	svr.lldpTxPktCh = make(chan SendPktChannel, LLDP_TX_PKT_CHANNEL_SIZE)
	//svr.packet = packet.NewPacketInfo()
	svr.lldpExit = make(chan bool)
	svr.lldpSnapshotLen = 1024
	svr.lldpPromiscuous = false
	// LLDP Notifications are atleast 5 seconds apart with default being
	// 30 seconds. So, we can have the leavrage the pcap timeout (read from
	// buffer) to be 1 second.
	svr.lldpTimeout = 1 * time.Second
}

/* De-Allocate memory to all the object which are being used by LLDP server
 */
func (svr *LLDPServer) DeInitGlobalDS() {
	svr.lldpRxPktCh = nil
	svr.lldpTxPktCh = nil
	svr.lldpGblInfo = nil
	svr.lldpPortNumIfIndexMap = nil
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

/*  lldp server: 1) Connect to all the clients
 *		 2) Initialize DB
 *		 3) Read from DB and close DB
 *		 5) go routine to handle all the channels within lldp server
 */
func (svr *LLDPServer) LLDPStartServer(paramsDir string) {
	svr.paramsDir = paramsDir
	// First connect to client to avoid any issues with start/re-start
	svr.ConnectAndInitPortVlan()

	// Initialize DB
	err := svr.InitDB()
	if err != nil {
		debug.Logger.Err("DB init failed")
	} else {
		// Populate Gbl Configs
		svr.ReadDB()
		svr.CloseDB()
	}
	go svr.ChannelHanlder()
}

/* lldp server go ahead and connect to asicd.. Support is there if lldp needs to
 * connect any other client like, lacp, stp, etc...
 */
func (svr *LLDPServer) ConnectAndInitPortVlan() error {
	configFile := svr.paramsDir + "/clients.json"
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Error while reading ",
			"configuration file", configFile))
		return err
	}
	var unConnectedClients []LLDPClientJson
	err = json.Unmarshal(bytes, &unConnectedClients)
	if err != nil {
		debug.Logger.Err("Error in Unmarshalling Json")
		return err
	}
	re_connect := 15
	count := 0
	// connect to client
	for {
		time.Sleep(time.Millisecond * 500)
		for i := 0; i < len(unConnectedClients); i++ {
			err := svr.ConnectToUnConnectedClient(
				unConnectedClients[i])
			if err == nil {
				debug.Logger.Info("Connected to " +
					unConnectedClients[i].Name)
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)

			} else if err.Error() ==
				LLDP_CLIENT_CONNECTION_NOT_REQUIRED {
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)
			} else {
				count++
				if count == re_connect {
					debug.Logger.Err(fmt.Sprintln("Connecting to",
						unConnectedClients[i].Name,
						"failed ERROR:", err))
					count = 0
				}
			}
		}
		if len(unConnectedClients) == 0 {
			break
		}
	}
	// This will do bulk get
	svr.GetInfoFromAsicd()
	// OS Signal channel listener thread
	svr.OSSignalHandle()
	return err
}

/*  Helper function specifying which clients lldp needs to connect
 *  if needed to connect to other client add case for it
 */
func (svr *LLDPServer) ConnectToUnConnectedClient(client LLDPClientJson) error {
	switch client.Name {
	case "asicd":
		return svr.ConnectToAsicd(client)
	default:
		return errors.New(LLDP_CLIENT_CONNECTION_NOT_REQUIRED)
	}
}

/*  Helper function to connect asicd
 */
func (svr *LLDPServer) ConnectToAsicd(client LLDPClientJson) error {
	var err error
	svr.asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
	svr.asicdClient.Transport, svr.asicdClient.PtrProtocolFactory, err =
		ipcutils.CreateIPCHandles(svr.asicdClient.Address)
	if svr.asicdClient.Transport == nil ||
		svr.asicdClient.PtrProtocolFactory == nil ||
		err != nil {
		return err
	}
	svr.asicdClient.ClientHdl =
		asicdServices.NewASICDServicesClientFactory(
			svr.asicdClient.Transport,
			svr.asicdClient.PtrProtocolFactory)
	svr.asicdClient.IsConnected = true
	return nil
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
						gblInfo.Name))
					continue
				}
				// reset/start timer for recipient information
				gblInfo.RxInfo.CheckPeerEntry(gblInfo.Name)
				svr.lldpGblInfo[rcvdInfo.ifIndex] = gblInfo
				// dump the frame
				//gblInfo.DumpFrame()
				/*
					// Cache the received incoming frame
					err := gblInfo.ProcessRxPkt(rcvdInfo.pkt)
					if err != nil {
						debug.Logger.Err(fmt.Sprintln("err", err,
							" while processing rx frame on port",
							gblInfo.Name))
						continue
					}
					// reset/start timer for recipient information
					gblInfo.CheckPeerEntry()
					svr.lldpGblInfo[rcvdInfo.ifIndex] = gblInfo
				*/
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
				gblInfo.SendFrame()
				svr.lldpGblInfo[info.ifIndex] = gblInfo
			}
		}
	}
}

/* Create l2 port global map..
 *	lldpGlbInfo will contain all the runtime information for lldp server
 */
func (svr *LLDPServer) InitL2PortInfo(portConf *asicdServices.PortState) {
	gblInfo, _ := svr.lldpGblInfo[portConf.IfIndex]
	gblInfo.InitRuntimeInfo(portConf)
	svr.lldpGblInfo[portConf.IfIndex] = gblInfo
	svr.lldpPortNumIfIndexMap[portConf.PortNum] = gblInfo.IfIndex
	svr.lldpIntfStateSlice = append(svr.lldpIntfStateSlice, gblInfo.IfIndex)
}

/*  Update l2 port info done via GetBulkPort() which will return port config
 *  information.. We will update each fpPort/ifindex with mac address of its own
 */
func (svr *LLDPServer) UpdateL2PortInfo(portConf *asicdServices.Port) {
	gblInfo, exists := svr.lldpGblInfo[svr.lldpPortNumIfIndexMap[portConf.PortNum]]
	if !exists {
		debug.Logger.Err(fmt.Sprintln("no mapping found for Port Num",
			portConf.PortNum, "and hence we do not have any MacAddr"))
		return
	}
	gblInfo.UpdatePortInfo(portConf)
	svr.lldpGblInfo[gblInfo.IfIndex] = gblInfo
	// Only start rx/tx once we have got the mac address from the get bulk port
	gblInfo.OperStateLock.RLock()
	if gblInfo.OperState == LLDP_PORT_STATE_UP {
		gblInfo.OperStateLock.RUnlock()
		svr.StartRxTx(gblInfo.IfIndex)
	} else {
		gblInfo.OperStateLock.RUnlock()
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
	gblInfo.CreatePcapHandler(svr.lldpSnapshotLen, svr.lldpPromiscuous,
		svr.lldpTimeout)

	svr.lldpGblInfo[ifIndex] = gblInfo
	debug.Logger.Info("Start lldp frames rx/tx for port:" + gblInfo.Name)
	go svr.ReceiveFrames(gblInfo.PcapHandle, ifIndex)
	go svr.TransmitFrames(gblInfo.PcapHandle, ifIndex)
	svr.lldpUpIntfStateSlice = append(svr.lldpUpIntfStateSlice,
		gblInfo.IfIndex)
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
	gblInfo.StopTxTimer()
	// delete pcap handler
	gblInfo.DeletePcapHandler()
	// invalid the cache information
	gblInfo.DeleteCacheFrame()
	debug.Logger.Info("Stop lldp frames rx/tx for port:" + gblInfo.Name)
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
