package lldpServer

import (
	_ "asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"io/ioutil"
	_ "net"
	"os"
	"os/signal"
	"strconv"
	_ "strings"
	"sync"
	"syscall"
	"time"
	"utils/ipcutils"
	"utils/logging"
)

func LLDPNewServer(log *logging.Writer) *LLDPServer {
	lldpServerInfo := &LLDPServer{}
	lldpServerInfo.logger = log
	// Allocate memory to all the Data Structures
	lldpServerInfo.LLDPInitGlobalDS()
	return lldpServerInfo
}

func (svr *LLDPServer) LLDPInitGlobalDS() {
	svr.lldpGblInfo = make(map[int32]LLDPGlobalInfo,
		LLDP_INITIAL_GLOBAL_INFO_CAPACITY)
	svr.lldpSnapshotLen = 1024
	svr.lldpPromiscuous = false
	svr.lldpTimeout = 10 * time.Microsecond
}

func (svr *LLDPServer) LLDPDeInitGlobalDS() {
	svr.lldpGblInfo = nil
}

func (svr *LLDPServer) LLDPCloseAllPcapHandlers() {
	for i := 0; i < len(svr.lldpIntfStateSlice); i++ {
		key := svr.lldpIntfStateSlice[i]
		gblInfo := svr.lldpGblInfo[key]
		if gblInfo.PcapHandle != nil {
			gblInfo.PcapHdlLock.Lock()
			gblInfo.PcapHandle.Close()
			gblInfo.PcapHdlLock.Unlock()
		}
	}
}

func (svr *LLDPServer) LLDPStartServer(paramsDir string) {
	svr.paramsDir = paramsDir
	// First connect to client to avoid any issues with start/re-start
	svr.LLDPConnectAndInitPortVlan()

	// Initialize DB
	err := svr.LLDPInitDB()
	if err != nil {
		svr.logger.Err("DB init failed")
	} else {
		// Populate Gbl Configs
		svr.LLDPReadDB()
		svr.LLDPCloseDB()
	}
	go svr.LLDPChannelHanlder()
}

func (svr *LLDPServer) LLDPConnectAndInitPortVlan() error {
	configFile := svr.paramsDir + "/clients.json"
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Error while reading configuration file",
			configFile))
		return err
	}
	var unConnectedClients []LLDPClientJson
	err = json.Unmarshal(bytes, &unConnectedClients)
	if err != nil {
		svr.logger.Err("Error in Unmarshalling Json")
		return err
	}

	// connect to client
	for {
		time.Sleep(time.Millisecond * 500)
		for i := 0; i < len(unConnectedClients); i++ {
			err := svr.LLDPConnectToUnConnectedClient(unConnectedClients[i])
			if err == nil {
				svr.logger.Info("Connected to " +
					unConnectedClients[i].Name)
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)

			} else if err.Error() == LLDP_CLIENT_CONNECTION_NOT_REQUIRED {
				svr.logger.Info("connection to " +
					unConnectedClients[i].Name +
					" not required")
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)
			}
		}
		if len(unConnectedClients) == 0 {
			svr.logger.Info("all clients connected successfully")
			break
		}
	}

	svr.LLDPGetInfoFromAsicd()

	// OS Signal channel listener thread
	svr.LLDPOSSignalHandle()
	return err
}

func (svr *LLDPServer) LLDPConnectToUnConnectedClient(client LLDPClientJson) error {
	switch client.Name {
	case "asicd":
		return svr.LLDPConnectToAsicd(client)
	default:
		return errors.New(LLDP_CLIENT_CONNECTION_NOT_REQUIRED)
	}
}

func (svr *LLDPServer) LLDPConnectToAsicd(client LLDPClientJson) error {
	var err error
	svr.asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
	svr.asicdClient.Transport, svr.asicdClient.PtrProtocolFactory, err =
		ipcutils.CreateIPCHandles(svr.asicdClient.Address)
	if svr.asicdClient.Transport == nil ||
		svr.asicdClient.PtrProtocolFactory == nil ||
		err != nil {
		svr.logger.Err(fmt.Sprintln("Connecting to",
			client.Name, "failed ", err))
		return err
	}
	svr.asicdClient.ClientHdl =
		asicdServices.NewASICDServicesClientFactory(
			svr.asicdClient.Transport,
			svr.asicdClient.PtrProtocolFactory)
	svr.asicdClient.IsConnected = true
	return nil
}

func (svr *LLDPServer) LLDPOSSignalHandle() {
	sigChannel := make(chan os.Signal, 1)
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChannel, signalList...)
	go svr.LLDPSignalHandler(sigChannel)
}

func (svr *LLDPServer) LLDPSignalHandler(sigChannel <-chan os.Signal) {
	signal := <-sigChannel
	switch signal {
	case syscall.SIGHUP:
		svr.logger.Alert("Received SIGHUP Signal")
		svr.LLDPCloseAllPcapHandlers()
		svr.LLDPDeInitGlobalDS()
		svr.logger.Alert("Closed all pcap's and freed memory")
		os.Exit(0)
	default:
		svr.logger.Info(fmt.Sprintln("Unhandled Signal:", signal))
	}
}

func (svr *LLDPServer) LLDPChannelHanlder() {
	// Start receviing in rpc values in the channell
}

func (svr *LLDPServer) LLDPInitL2PortInfo(portConf *asicdServices.PortState) {
	gblInfo, _ := svr.lldpGblInfo[portConf.IfIndex]
	gblInfo.IfIndex = portConf.IfIndex
	gblInfo.Name = portConf.Name
	gblInfo.OperState = portConf.OperState
	gblInfo.PortNum = portConf.PortNum
	gblInfo.OperStateLock = &sync.RWMutex{}
	gblInfo.PcapHdlLock = &sync.RWMutex{}
	svr.lldpGblInfo[portConf.IfIndex] = gblInfo
	if gblInfo.OperState == LLDP_PORT_STATE_UP {
		svr.LLDPCreatePcapHandler(gblInfo.IfIndex)
	}
	svr.lldpIntfStateSlice = append(svr.lldpIntfStateSlice, gblInfo.IfIndex)
	svr.logger.Info("Port " + gblInfo.Name + " is " + gblInfo.OperState)
}

func (svr *LLDPServer) LLDPCreatePcapHandler(ifIndex int32) {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		svr.logger.Err(fmt.Sprintln("No entry for ifindex", ifIndex))
		return
	}
	pcapHdl, err := pcap.OpenLive(gblInfo.Name, svr.lldpSnapshotLen,
		svr.lldpPromiscuous, svr.lldpTimeout)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Creating Pcap Handler failed for",
			gblInfo.Name, "Error:", err))
	}
	err = pcapHdl.SetBPFFilter(LLDP_BPF_FILTER)
	if err != nil {
		svr.logger.Info(fmt.Sprintln("setting filter", LLDP_BPF_FILTER,
			"for", gblInfo.Name, "failed with error:", err))
	}
	gblInfo.PcapHdlLock.Lock()
	gblInfo.PcapHandle = pcapHdl
	gblInfo.PcapHdlLock.Unlock()
	svr.lldpGblInfo[ifIndex] = gblInfo
}
