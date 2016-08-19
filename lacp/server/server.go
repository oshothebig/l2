package server

import (
	"asicd/asicdCommonDefs"
	"fmt"
	"l2/lacp/protocol/drcp"
	"l2/lacp/protocol/lacp"
	"l2/lacp/protocol/utils"
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
	LAConfigMsgAggregatorCreated
	LAConfigMsgCreateConversationId
	LAConfigMsgUpdateConversationId
	LAConfigMsgDeleteConversationId
	// events based on port being in distributed state or not
	LAConfigMsgLaPortChannelDistributedRelayPortListUpdate
)

type LAConfig struct {
	Msgtype LaConfigMsgType
	Msgdata interface{}
}

type LAServer struct {
	logger           *logging.Writer
	ConfigCh         chan LAConfig
	AsicdSubSocketCh chan commonDefs.AsicdNotifyMsg
}

func NewLAServer(logger *logging.Writer) *LAServer {
	return &LAServer{
		logger:           logger,
		ConfigCh:         make(chan LAConfig),
		AsicdSubSocketCh: make(chan commonDefs.AsicdNotifyMsg),
	}
}

func (server *LAServer) InitServer() {
	utils.ConstructPortConfigMap()
	drcp.GetAllCVIDConversations()
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
		drcp.AttachAggregatorToDistributedRelay(config.Id)

	case LAConfigMsgDeleteLaPortChannel:
		s.logger.Info("CONFIG: Delete Link Aggregation Group / Port Channel")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		drcp.DetachAggregatorFromDistributedRelay(config.Id)
		lacp.DeleteLaAgg(config.Id)

	case LAConfigMsgUpdateLaPortChannelLagHash:
		s.logger.Info("CONFIG: Link Aggregation Group / Port Channel Lag Hash Mode")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		lacp.SetLaAggHashMode(config.Id, config.HashMode)

	case LAConfigMsgUpdateLaPortChannelSystemIdMac:
		s.logger.Info("CONFIG: Link Aggregation Group / Port Channel SystemId MAC")
		config := conf.Msgdata.(*lacp.LaAggConfig)
		var a *lacp.LaAggregator
		if lacp.LaFindAggById(config.Id, &a) {
			// configured ports
			for _, pId := range a.PortNumList {
				lacp.SetLaAggPortSystemInfo(uint16(pId), config.Lacp.SystemIdMac, config.Lacp.SystemPriority)
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
		if lacp.LaFindAggById(config.Id, &a) {

			if config.Type == lacp.LaAggTypeSTATIC {
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
		var a *lacp.LaAggregator
		// if this is the first port then we need to do some
		// work in terms of the
		drcp.UpdateAggregatorPortList(config.AggId)

	case LAConfigMsgDeleteLaAggPort:
		s.logger.Info("CONFIG: Delete Link Aggregation Port")
		config := conf.Msgdata.(*lacp.LaAggPortConfig)
		lacp.DeleteLaAggPort(config.Id)

	case LAConfigMsgCreateDistributedRelay:
		s.logger.Info("CONFIG: Create Distributed Relay")
		config := conf.Msgdata.(*drcp.DistrubtedRelayConfig)
		drcp.CreateDistributedRelay(config)

	case LAConfigMsgDeleteDistributedRelay:
		s.logger.Info("CONFIG: Delete Distributed Relay")
		config := conf.Msgdata.(*drcp.DistrubtedRelayConfig)
		drcp.DeleteDistributedRelay(config.GetKey())

	case LAConfigMsgCreateConversationId:
		s.logger.Info("CONFIG: Create Conversation Id")
		config := conf.Msgdata.(*drcp.DRConversationConfig)
		drcp.CreateConversationId(config)

	case LAConfigMsgDeleteConversationId:
		s.logger.Info("CONFIG: Delete Conversation Id")
		config := conf.Msgdata.(*drcp.DRConversationConfig)
		drcp.DeleteConversationId(config)

	case LAConfigMsgUpdateConversationId:
		s.logger.Info("CONFIG: Update Conversation Id")
		config := conf.Msgdata.(*drcp.DRConversationConfig)
		drcp.UpdateConversationId(config)

	case LAConfigMsgLaPortChannelDistributedRelayPortListUpdate:
		s.logger.Info("CONFIG: Update DR AGG Port List")
		config := conf.Msgdata.(*drcp.DRAggregatorPortListConfig)
		drcp.DistributedRelayAggregatorPortListUpdate(config)
	}
}

func (s *LAServer) processLinkDownEvent(linkId int) {
	s.logger.Info(fmt.Sprintln("LA EVT: Link Down", linkId))
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(linkId), &p) {
		p.LaAggPortDisable()
		p.LinkOperStatus = false
	} else {
		for _, ipp := range drcp.DRCPIppDBList {
			if int(ipp.Id) == linkId {
				go ipp.DrIppLinkDown()
			}
		}
	}
}

func (s *LAServer) processLinkUpEvent(linkId int) {
	s.logger.Info(fmt.Sprintln("LA EVT: Link Up", linkId))
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(linkId), &p) {
		p.LaAggPortEnabled()
		p.LinkOperStatus = true
	} else {
		for _, ipp := range drcp.DRCPIppDBList {
			if int(ipp.Id) == linkId {
				go ipp.DrIppLinkUp()
			}
		}
	}
}

func (s *LAServer) processVlanEvent(vlanMsg commonDefs.VlanNotifyMsg) {

	// need to determine whether the message is a new vlan or if
	// ports were updated are any of them part of any of the
	// aggregators which exist
	msgtype := LAConfigMsgUpdateConversationId
	if vlanMsg.MsgType == commonDefs.NOTIFY_VLAN_CREATE {
		msgtype = LAConfigMsgCreateConversationId
	} else if vlanMsg.MsgType == commonDefs.NOTIFY_VLAN_DELETE {
		msgtype = LAConfigMsgDeleteConversationId
	}
	var agg *lacp.LaAggregator
	for lacp.LaGetAggNext(&agg) {
		portList := vlanMsg.TagPorts
		for _, p := range vlanMsg.UntagPorts {
			portList = append(portList, p)
		}

		for _, uifindex := range portList {
			id := asicdCommonDefs.GetIntfIdFromIfIndex(uifindex)
			for _, aggport := range agg.PortNumList {
				if id == int(aggport) {
					var dr *drcp.DistributedRelay
					if drcp.DrFindByAggregator(int32(agg.AggId), &dr) {
						// gateway message
						cfg := drcp.DRConversationConfig{
							DrniName: dr.DrniName,
							Idtype:   drcp.GATEWAY_ALGORITHM_CVID,
							Cvlan:    uint16(vlanMsg.VlanId),
						}

						s.ConfigCh <- LAConfig{
							Msgtype: msgtype,
							Msgdata: cfg,
						}
					}
				}
			}
		}
	}
}

/*

type LagNotifyMsg struct {
	MsgType     uint8
	LagName     string
	IfIndex     int32
	IfIndexList []int32
}

*/
func (s *LAServer) processLAGEvent(lagMsg commonDefs.LagNotifyMsg) {

	// Only care about port updates
	if lagMsg.MsgType == commonDefs.NOTIFY_LAG_UPDATE {
		s.logger.Info(fmt.Sprintln("Msg lag = ", lagMsg))
		msgtype := LAConfigMsgLaPortChannelDistributedRelayPortListUpdate
		aggId := asicdCommonDefs.GetIntfIdFromIfIndex(lagMsg.IfIndex)
		cfg := drcp.DRAggregatorPortListConfig{
			DrniAggregator: uint32(aggId),
		}
		for _, ifindex := range lagMsg.IfIndexList {
			cfg.PortList = append(cfg.PortList, int32(asicdCommonDefs.GetIntfIdFromIfIndex(ifindex)))
		}

		s.ConfigCh <- LAConfig{
			Msgtype: msgtype,
			Msgdata: cfg,
		}
	}
}

func (s *LAServer) processAsicdNotification(msg commonDefs.AsicdNotifyMsg) {
	switch msg.(type) {
	case commonDefs.L2IntfStateNotifyMsg:
		l2Msg := msg.(commonDefs.L2IntfStateNotifyMsg)
		s.logger.Info(fmt.Sprintf("Msg linkstatus = %d msg port = %d\n", l2Msg.IfState, l2Msg.IfIndex))
		if l2Msg.IfState == asicdCommonDefs.INTF_STATE_DOWN {
			s.processLinkDownEvent(asicdCommonDefs.GetIntfIdFromIfIndex(l2Msg.IfIndex)) //asicd always sends out link State events for PHY ports
		} else {
			s.processLinkUpEvent(asicdCommonDefs.GetIntfIdFromIfIndex(l2Msg.IfIndex))
		}
	case commonDefs.VlanNotifyMsg:
		vlanMsg := msg.(commonDefs.VlanNotifyMsg)
		s.logger.Info(fmt.Sprintln("Msg vlan = ", vlanMsg))
		s.processVlanEvent(vlanMsg)

	case commonDefs.LagNotifyMsg:
		lagMsg := msg.(commonDefs.LagNotifyMsg)
		s.processLAGEvent(lagMsg)
	}
}
