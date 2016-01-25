// port.go
package stp

import (
	"time"
)

type StpPort struct {
	// 17.19
	AgeingTime         int32
	Agree              bool
	Agreed             bool
	DesignatedPriority PriorityVector
	DesignatedTimes    Times
	Disputed           bool
	FdbFlush           bool
	Forward            bool
	Forwarding         bool
	Infols             PortInfoState
	Learn              bool
	Learning           bool
	Mcheck             bool
	MsgPriority        PriorityVector
	MsgTimes           Times
	NewInfo            bool
	OperEdge           bool
	PortEnabled        bool
	PortId             int32
	PortPathCost       int32
	PortPriority       PriorityVector
	PortTimes          Times
	Proposed           bool
	Proposing          bool
	RcvdBPDU           bool
	RcvdfInfo          PortDesignatedRcvInfo
	RcvdMsg            bool
	RcvdRSTP           bool
	RcvdSTP            bool
	RcvdTc             bool
	RcvdTcAck          bool
	RcvdTcn            bool
	ReRoot             bool
	Reselect           bool
	Role               PortRole
	Selected           bool
	SelectedRole       PortRole
	SendRSTP           bool
	Sync               bool
	Synced             bool
	TcAck              bool
	TcProp             bool
	Tick               bool
	TxCount            uint64
	UpdtInfo           bool

	// 17.17
	EdgeDelayWhileTimer PortTimer
	FdWhileTimer        PortTimer
	HelloWhenTimer      PortTimer
	MdelayWhiletimer    PortTimer
	RbWhileTimer        PortTimer
	RcvdInfoWhiletimer  PortTimer
	RrWhileTimer        PortTimer
	TcWhileTimer        PortTimer

	PortTimerMap map[TimerType]*PortTimer
}

type PortTimer struct {
	t     *time.Timer
	d     time.Duration
	count int32
}

func NewStpPort(c *StpPortConfig) {

	p := &StpPort{
		PortTimerMap: make(map[TimerType]*PortTimer),
	}
	p.PortTimerMap[TimerTypeEdgeDelayWhile] = &p.EdgeDelayWhileTimer
	p.PortTimerMap[TimerTypeFdWhile] = &p.FdWhileTimer
	p.PortTimerMap[TimerTypeHelloWhen] = &p.HelloWhenTimer
	p.PortTimerMap[TimerTypeMdelayWhile] = &p.MdelayWhiletimer
	p.PortTimerMap[TimerTypeRbWhile] = &p.RbWhileTimer
	p.PortTimerMap[TimerTypeRcvdInfoWhile] = &p.RcvdInfoWhiletimer
	p.PortTimerMap[TimerTypeRrWhile] = &p.RrWhileTimer
	p.PortTimerMap[TimerTypeTcWhile] = &p.TcWhileTimer

}

func (p *StpPort) ResetCounter(counterType TimerType) {

}

func (p *StpPort) DecramentCounter(counterType TimerType) {
	switch counterType {
	case TimerTypeEdgeDelayWhile:
		p.EdgeDelayWhileTimer.count--
	case TimerTypeFdWhile:
		p.FdWhileTimer.count--
	case TimerTypeHelloWhen:
		p.HelloWhenTimer.count--
	case TimerTypeMdelayWhile:
		p.MdelayWhiletimer.count--
	case TimerTypeRbWhile:
		p.RbWhileTimer.count--
	case TimerTypeRcvdInfoWhile:
		p.RcvdInfoWhiletimer.count--
	case TimerTypeRrWhile:
		p.RrWhileTimer.count--
	case TimerTypeTcWhile:
		p.TcWhileTimer.count--
	}
}
