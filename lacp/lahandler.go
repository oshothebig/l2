// lahandler
package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	lacp "l2/lacp/protocol"
	"lacpd"
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
	Mode lacpd.Int,
	Timeout lacpd.Int, //time.Duration
	Mac string,
	Speed lacpd.Int,
	Duplex lacpd.Int,
	Mtu lacpd.Int,
	SysId string,
	IntfId string) (lacpd.Int, error) {

	fmt.Println("Thrift: CreateLaAggPort: Start")
	t := lacp.LacpShortTimeoutTime
	if Timeout == 90 {
		t = lacp.LacpLongTimeoutTime

	}
	conf := &lacp.LaAggPortConfig{
		Id:      uint16(Id),
		Prio:    uint16(Prio),
		AggId:   int(AggId),
		Key:     uint16(Key),
		Enable:  Enable,
		Mode:    int(Mode),
		Timeout: t,
		Properties: lacp.PortProperties{
			Mac:    ConvertStringToUint8Array(Mac),
			Speed:  int(Speed),
			Duplex: int(Duplex),
			Mtu:    int(Mtu),
		},
		TraceEna: false,
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
	return 1, errors.New("Unable to find Port")
}
