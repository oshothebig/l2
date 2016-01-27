// timers
package stp

import (
	"time"
)

type TimerType int

const (
	TimerTypeEdgeDelayWhile TimerType = iota
	TimerTypeFdWhile
	TimerTypeHelloWhen
	TimerTypeMdelayWhile
	TimerTypeRbWhile
	TimerTypeRcvdInfoWhile
	TimerTypeRrWhile
	TimerTypeTcWhile
)

var TimerTypeStrMap map[TimerType]string

func TimerTypeStrStateMapInit() {
	TimerTypeStrMap = make(map[TimerType]string)
	TimerTypeStrMap[TimerTypeEdgeDelayWhile] = "Edge Delay Timer"
	TimerTypeStrMap[TimerTypeFdWhile] = "Fd While Timer"
	TimerTypeStrMap[TimerTypeHelloWhen] = "Hello When Timer"
	TimerTypeStrMap[TimerTypeMdelayWhile] = "Migration Delay Timer"
	TimerTypeStrMap[TimerTypeRbWhile] = "Recent Backup Timer"
	TimerTypeStrMap[TimerTypeRcvdInfoWhile] = "Received Info Timer"
	TimerTypeStrMap[TimerTypeRrWhile] = "Recent Root Timer"
	TimerTypeStrMap[TimerTypeTcWhile] = "topology Change Timer"
}

// TickTimerStart: Port Timers Tick timer
func (m *PtmMachine) TickTimerStart() {

	if m.TickTimer == nil {
		m.TickTimer = time.NewTimer(time.Second * 1)
	} else {
		m.TickTimer.Reset(time.Second * 1)
	}
}

// TickTimerStop
// Stop the running timer
func (m *PtmMachine) TickTimerStop() {

	if m.TickTimer != nil {
		m.TickTimer.Stop()
	}
}

func (m *PtmMachine) TickTimerDestroy() {
	m.TickTimerStop()
	m.TickTimer = nil
}
