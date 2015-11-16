// lahandler
package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	lacp "l2/lacp/protocol"
	"lacpd"
	"strings"
	"time"
)

type LaServiceHandler struct {
}

func NewLaServiceHandler() *LaServiceHandler {
	return &LaServiceHandler{}
}

func ConvertStringToUint8Array(s string) [6]uint8 {
	var arr [6]uint8
	x, _ := hex.DecodeString(s)
	for k, v := range x {
		arr[k] = uint8(v)
	}
	return arr
}

func (la LaServiceHandler) CreateLaAggregator(aggId lacpd.Int,
	actorAdminKey lacpd.Uint16,
	actorSystemId string,
	aggMacAddr string) (lacpd.Int, error) {

	conf := &lacp.LaAggConfig{
		// Aggregator_MAC_address
		Mac: ConvertStringToUint8Array(aggMacAddr),
		// Aggregator_Identifier
		Id: int(aggId),
		// Actor_Admin_Aggregator_Key
		Key: uint16(actorAdminKey),
		// system to attach this agg to
		SysId: ConvertStringToUint8Array(actorSystemId),
	}

	lacp.CreateLaAgg(conf)
	return 0, nil
}

func (la LaServiceHandler) DeleteLaAggregator(aggId lacpd.Int) (lacpd.Int, error) {

	lacp.DeleteLaAgg(int(aggId))
	return 0, nil
}

func (la LaServiceHandler) CreateLaAggPort(Id lacpd.Uint16,
	Prio lacpd.Uint16,
	Key lacpd.Uint16,
	AggId lacpd.Int,
	Enable bool,
	Mode string, //"ON", "ACTIVE", "PASSIVE"
	Timeout string, //"SHORT", "LONG"
	Mac string,
	Speed lacpd.Int,
	Duplex lacpd.Int,
	Mtu lacpd.Int,
	SysId string,
	IntfId string) (lacpd.Int, error) {

	// TODO: check syntax on strings?
	t := lacp.LacpShortTimeoutTime
	if Timeout == "LONG" {
		t = lacp.LacpLongTimeoutTime
	}
	m := lacp.LacpModeOn
	if Mode == "ACTIVE" {
		m = lacp.LacpModeActive
	} else if Mode == "PASSIVE" {
		m = lacp.LacpModePassive
	}

	conf := &lacp.LaAggPortConfig{
		Id:      uint16(Id),
		Prio:    uint16(Prio),
		AggId:   int(AggId),
		Key:     uint16(Key),
		Enable:  Enable,
		Mode:    m,
		Timeout: t,
		Properties: lacp.PortProperties{
			Mac:    ConvertStringToUint8Array(Mac),
			Speed:  int(Speed),
			Duplex: int(Duplex),
			Mtu:    int(Mtu),
		},
		TraceEna: true,
		IntfId:   IntfId,
		SysId:    ConvertStringToUint8Array(SysId),
	}
	lacp.CreateLaAggPort(conf)
	return 0, nil
}

func (la LaServiceHandler) DeleteLaAggPort(Id lacpd.Uint16) (lacpd.Int, error) {
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(Id), &p) {
		if p.AggId != 0 {
			lacp.DeleteLaAggPortFromAgg(p.AggId, uint16(Id))
		}
		lacp.DeleteLaAggPort(uint16(Id))
		return 0, nil
	}
	return 1, errors.New(fmt.Sprintf("LACP: Unable to find Port %d", Id))
}

func (la LaServiceHandler) SetLaAggPortMode(Id lacpd.Uint16, m string) (lacpd.Int, error) {

	supportedModes := map[string]int{"ON": lacp.LacpModeOn,
		"ACTIVE":  lacp.LacpModeActive,
		"PASSIVE": lacp.LacpModePassive}

	if mode, ok := supportedModes[m]; ok {
		var p *lacp.LaAggPort
		if lacp.LaFindPortById(uint16(Id), &p) {
			lacp.SetLaAggPortLacpMode(uint16(Id), mode, p.TimeoutGet())
		} else {
			return 1, errors.New(fmt.Sprintf("LACP: Mode not set,  Unable to find Port", Id))
		}
	} else {
		return 1, errors.New(fmt.Sprintf("LACP: Mode not set,  Unsupported mode %d", m, "supported modes: ", supportedModes))
	}

	return 0, nil
}

func (la LaServiceHandler) SetLaAggPortTimeout(Id lacpd.Uint16, t string) (lacpd.Int, error) {
	// user can supply the periodic time or the
	supportedTimeouts := map[string]time.Duration{"SHORT": lacp.LacpShortTimeoutTime,
		"LONG": lacp.LacpLongTimeoutTime}

	if timeout, ok := supportedTimeouts[t]; ok {
		var p *lacp.LaAggPort
		if lacp.LaFindPortById(uint16(Id), &p) {
			lacp.SetLaAggPortLacpMode(uint16(Id), p.ModeGet(), timeout)
		} else {
			return 1, errors.New(fmt.Sprintf("LACP: timeout not set,  Unable to find Port", Id))
		}
	} else {
		return 1, errors.New(fmt.Sprintf("LACP: Timeout not set,  Unsupported timeout %d", t, "supported modes: ", supportedTimeouts))
	}

	return 0, nil
}

// SetPortLacpLogEnable will enable on a per port basis logging
// modStr - PORT, RXM, TXM, PTXM, TXM, CDM, ALL
// modStr can be a string containing one or more of the above
func (la LaServiceHandler) SetPortLacpLogEnable(Id lacpd.Uint16, modStr string, ena bool) (lacpd.Int, error) {
	modules := make(map[string]chan bool)
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(Id), &p) {
		modules["RXM"] = p.RxMachineFsm.RxmLogEnableEvent
		modules["TXM"] = p.TxMachineFsm.TxmLogEnableEvent
		modules["PTXM"] = p.PtxMachineFsm.PtxmLogEnableEvent
		modules["TXM"] = p.TxMachineFsm.TxmLogEnableEvent
		modules["CDM"] = p.CdMachineFsm.CdmLogEnableEvent
		modules["MUXM"] = p.MuxMachineFsm.MuxmLogEnableEvent

		for k, v := range modules {
			if strings.Contains(k, "PORT") || strings.Contains(k, "ALL") {
				p.EnableLogging(ena)
			}
			if strings.Contains(k, modStr) || strings.Contains(k, "ALL") {
				v <- ena
			}
		}
		return 0, nil
	}
	return 1, errors.New(fmt.Sprintf("LACP: LOG set failed,  Unable to find Port", Id))
}
