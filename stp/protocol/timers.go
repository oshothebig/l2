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

// WaitWhileTimerStart
// Start the timer
func (m *PtmMachine) TimerStart() {
	p := m.p

	t := p.PortTimerMap[m.MachineTimerType].t
	d := p.PortTimerMap[m.MachineTimerType].d

	if t == nil {
		t = time.NewTimer(d)
	} else {
		t.Reset(d)
	}
}

// WaitWhileTimerStop
// Stop the timer, which should only happen
// on creation as well as when the lacp mode is "on"
func (m *PtmMachine) TimerStop() {
	p := m.p

	if p.PortTimerMap[m.MachineTimerType].t != nil {
		p.PortTimerMap[m.MachineTimerType].t.Stop()
	}
}

func (m *PtmMachine) TimerDurationSet(timeout time.Duration) {
	p := m.p
	p.PortTimerMap[m.MachineTimerType].d = timeout
}

func (m *PtmMachine) TimerDestroy() {
	p := m.p
	m.TimerStop()
	p.PortTimerMap[m.MachineTimerType].t = nil
}
