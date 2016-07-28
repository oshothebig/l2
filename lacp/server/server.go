package server

import (
	"asicd/asicdCommonDefs"
	"fmt"
	"l2/lacp/protocol/drcp"
	"l2/lacp/protocol/lacp"
	"utils/commonDefs"
	"utils/logging"
)

type LaConfigMsgType int8

const (
	LAConfigMsgCreateLaPortChannel LaConfigMsgType = iota + 1
	LAConfigMsgDeleteLaPortChannel
	LAConfigMsgUpdateLaPortChannelLagHash
	LAConfigMsgUpdateLaPortChannelSystemIdMac
	LAConfigMsgUpdateLaPortChannelSystemPriority
	LAConfigMsgUpdateLaPortChannelLagType
	LAConfigMsgUpdateLaPortChannelAdminState
	LAConfigMsgUpdateLaPortChannelAggMode
	LAConfigMsgUpdateLaPortChannelPeriod
	LAConfigMsgCreateLaAggPort
	LAConfigMsgDeleteLaAggPort
	LAConfigMsgUpdateLaAggPortAdminState
	LAConfigMsgCreateDistributedRelay
	LAConfigMsgDeleteDistributedRelay
)

type LAConfig struct {
	Msgtype LAConfigMsgType
	Msgdata interface{}
}

type LAServer struct {
	logger           *logging.Writer
	ConfigCh         chan LAConfig
	AsicdSubSocketCh chan commonDefs.AsicdNotifyMsg
}

func NewLAServer(logger *logging.Writer) *STPServer {
	return &LAServer{
		logger:           logger,
		ConfigCh:         make(chan LAConfig),
		AsicdSubSocketCh: make(chan commonDefs.AsicdNotifyMsg),
	}
}

func (server *LAServer) InitServer() {
	utils.ConstructPortConfigMap()
}

// StartSTPSConfigNotificationListener
func (s *LAServer) StartLaConfigNotificationListener() {
	s.InitServer()
	//server.InitDone <- true
	go func(svr *LAServer) {
		svr.logger.Info("Starting LA Config Event Listener")
		for {
			select {
			case laConf, ok := <-svr.ConfigCh:
				if ok {
					svr.processLaConfig(laConf)
				} else {
					// channel was closed
					return
				}
			case msg := <-svr.AsicdSubSocketCh:
				svr.processAsicdNotification(msg)
			}
		}
	}(s)
}

func (s *LAServer) processLaConfig(conf LAConfig) {

	switch conf.Msgtype {
	case LAConfigMsgCreateLaPortChannel:
		s.logger.Info("CONFIG: Create Link Aggregation Group / Port Channel")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		lacp.CreateLaAgg(config)

	case LAConfigMsgDeleteLaPortChannel:
		s.logger.Info("CONFIG: Delete Link Aggregation Group / Port Channel")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		lacp.DeleteLaAgg(config.Id)

	case LAConfigMsgUpdateLaPortChannelLagHash:
		s.logger.Info("CONFIG: Link Aggregation Group / Port Channel Lag Hash Mode")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		lacp.SetLaAggHashMode(config.Id, conf.HashMode)

	case LAConfigMsgUpdateLaPortChannelSystemIdMac:
		s.logger.Info("CONFIG: Link Aggregation Group / Port Channel SystemId MAC")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		var a *lacp.LaAggregator
		if lacp.LaFindAggById(config.Id, &a) {
			// configured ports
			for _, pId := range a.PortNumList {
				lacp.SetLaAggPortSystemInfo(uint16(pId), conf.Lacp.SystemIdMac, conf.Lacp.SystemPriority)
			}
		}

	case LAConfigMsgUpdateLaPortChannelSystemPriority:
		s.logger.Info("CONFIG: Link Aggregation Group / Port Channel System Priority")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		var a *lacp.LaAggregator
		if lacp.LaFindAggById(config.Id, &a) {
			// configured ports
			for _, pId := range a.PortNumList {
				lacp.SetLaAggPortSystemInfo(uint16(pId), config.Lacp.SystemIdMac, config.Lacp.SystemPriority)
			}
		}

	case LAConfigMsgUpdateLaPortChannelLagType, LAConfigMsgUpdateLaPortChannelAggMode:
		s.logger.Info("CONFIG: Link Aggregation Group / Port Channel System Lag Type")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		var a *lacp.LaAggregator
		var p *lacp.LaAggPort
		if lacp.LaFindAggById(conf.Id, &a) {

			if conf.Type == lacp.LaAggTypeSTATIC {
				// configured ports
				for _, pId := range a.PortNumList {
					if lacp.LaFindPortById(uint16(pId), &p) {
						lacp.SetLaAggPortLacpMode(uint16(pId), lacp.LacpModeOn)
					}
				}
			} else {
				for _, pId := range a.PortNumList {
					if lacp.LaFindPortById(uint16(pId), &p) {
						lacp.SetLaAggPortLacpMode(uint16(pId), int(config.Lacp.Mode))
					}
				}
			}
		}

	case LAConfigMsgUpdateLaPortChannelAdminState:
		s.logger.Info("CONFIG: Link Aggregation Group / Port Channel System Lag Type")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		if config.Enabled {
			lacp.EnableLaAgg(config.Id)
		} else {
			lacp.DisableLaAgg(config.Id)
		}
	case LAConfigMsgUpdateLaPortChannelPeriod:
		s.logger.Info("CONFIG: Link Aggregation Group / Port Channel System Period")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		var a *lacp.LaAggregator
		if lacp.LaFindAggById(config.Id, &a) {
			// configured ports
			for _, pId := range a.PortNumList {
				lacp.SetLaAggPortLacpPeriod(uint16(pId), config.Lacp.Interval)
			}
		}
	case LAConfigMsgCreateLaAggPort:
		s.logger.Info("CONFIG: Create Link Aggregation Port")
		config := conf.Msgdata.(*lacp.LaAggPortConfig)
		lacp.CreateLaAggPort(config)

	case LAConfigMsgDeleteLaAggPort:
		s.logger.Info("CONFIG: Delete Link Aggregation Port")
		config := conf.Msgdata.(*lacp.LaAggPortConfig)
		lacp.DeleteLaAggPort(config.Id)
	}
}

func processLinkDownEvent(linkId int) {
	s.logger.Info(fmt.Sprintln("LA EVT: Link Down", linkId))
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(linkId), &p) {
		p.LaAggPortDisable()
		p.LinkOperStatus = false
	}
}

func processLinkUpEvent(linkId int) {
	s.logger.Info(fmt.Sprintln("LA EVT: Link Up", linkId))
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(linkId), &p) {
		p.LaAggPortEnabled()
		p.LinkOperStatus = true
	}
}

func (s *LAServer) processAsicdNotification(msg commonDefs.AsicdNotifyMsg) {
	switch msg.(type) {
	case commonDefs.L2IntfStateNotifyMsg:
		l2Msg := msg.(commonDefs.L2IntfStateNotifyMsg)
		s.logger.Info("Msg linkstatus = %d msg port = %d\n", l2Msg.IfState, l2Msg.IfIndex)
		if l2Msg.IfState == asicdCommonDefs.INTF_STATE_DOWN {
			processLinkDownEvent(asicdCommonDefs.GetIntfIdFromIfIndex(l2Msg.IfIndex)) //asicd always sends out link State events for PHY ports
		} else {
			processLinkUpEvent(asicdCommonDefs.GetIntfIdFromIfIndex(l2Msg.IfIndex))
		}
	}
}
