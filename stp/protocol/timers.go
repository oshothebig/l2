// timers
package stp

import (
	//"fmt"
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

func (p *StpPort) ResetTimerCounters(counterType TimerType) {
	// TODO
}

func (p *StpPort) DecrementTimerCounters() {
	/*StpMachineLogger("INFO", "PTIM", p.IfIndex, fmt.Sprintf("EdgeDelayWhile[%d] FdWhileTimer[%d] HelloWhen[%d] MdelayWhile[%d] RbWhile[%d] RcvdInfoWhile[%d] RrWhile[%d] TcWhile[%d] txcount[%d]",
	p.EdgeDelayWhileTimer.count,
	p.FdWhileTimer.count,
	p.HelloWhenTimer.count,
	p.MdelayWhiletimer.count,
	p.RbWhileTimer.count,
	p.RcvdInfoWhiletimer.count,
	p.RrWhileTimer.count,
	p.TcWhileTimer.count,
	p.TxCount))*/
	// 17.19.44
	if p.TxCount > 0 {
		p.TxCount--
	}

	// ed owner
	if p.EdgeDelayWhileTimer.count > 0 {
		p.EdgeDelayWhileTimer.count--
	}
	if p.EdgeDelayWhileTimer.count == 0 {
		defer p.NotifyEdgeDelayWhileTimerExpired()
	}

	// Prt owner
	if p.FdWhileTimer.count > 0 {
		if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDisabledPort &&
			uint16(p.FdWhileTimer.count) == p.PortTimes.MaxAge &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileNotEqualMaxAgeAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		}
		p.FdWhileTimer.count--
	}
	if p.FdWhileTimer.count == 0 {
		defer p.NotifyFdWhileTimerExpired()
	}
	// ptx owner
	if p.HelloWhenTimer.count > 0 {
		p.HelloWhenTimer.count--
	}
	if p.HelloWhenTimer.count == 0 {
		defer p.NotifyHelloWhenTimerExpired()
	}

	// ppm owner
	if p.MdelayWhiletimer.count > 0 {
		p.MdelayWhiletimer.count--
	}
	if p.MdelayWhiletimer.count == 0 {
		defer p.NotifyMdelayWhileTimerExpired()
	}
	// prt owner
	if p.RbWhileTimer.count > 0 {
		// this case should reset the rbwhiletimer
		if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateAlternatePort &&
			p.Role == PortRoleBackupPort &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventRbWhileNotEqualTwoTimesHelloTimeAndRoleEqualsBackupPortAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		}
		p.RbWhileTimer.count--
	}
	if p.RbWhileTimer.count == 0 {
		defer p.NotifyRbWhileTimerExpired()
	}
	// pi owner
	if p.RcvdInfoWhiletimer.count > 0 {
		p.RcvdInfoWhiletimer.count--
	}
	if p.RcvdInfoWhiletimer.count == 0 {
		defer p.NotifyRcvdInfoWhileTimerExpired()
	}
	// prt owner
	if p.RrWhileTimer.count > 0 {
		if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateRootPort {
			if p.RrWhileTimer.count != int32(p.b.RootTimes.ForwardingDelay) &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventRrWhileNotEqualFwdDelayAndSelectedAndNotUpdtInfo,
					src: PrtMachineModuleStr,
				}
			}
		}

		p.RrWhileTimer.count--
		if p.RrWhileTimer.count != 0 &&
			p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
			if p.ReRoot &&
				!p.OperEdge &&
				p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
					src: PrtMachineModuleStr,
				}
			} else if p.ReRoot &&
				!p.OperEdge &&
				p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
					src: PrtMachineModuleStr,
				}
			}
		}
	}
	if p.RrWhileTimer.count == 0 {
		defer p.NotifyRrWhileTimerExpired()
	}
	// tc owner
	if p.TcWhileTimer.count > 0 {
		p.TcWhileTimer.count--
	}
	if p.TcWhileTimer.count == 0 {
		defer p.NotifyTcWhileTimerExpired()
	}
}

func (p *StpPort) NotifyEdgeDelayWhileTimerExpired() {

	if p.BdmMachineFsm.Machine.Curr.CurrentState() == BdmStateNotEdge &&
		p.AutoEdgePort &&
		p.SendRSTP &&
		p.Proposing {
		p.BdmMachineFsm.BdmEvents <- MachineEvent{
			e:   BdmEventEdgeDelayWhileEqualZeroAndAutoEdgeAndSendRSTPAndProposing,
			src: BdmMachineModuleStr,
		}

	}
}

func (p *StpPort) NotifyFdWhileTimerExpired() {

	if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateRootPort {
		// events from Figure 17-21
		if p.RstpVersion &&
			!p.Learn &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo,
				src: PtmMachineModuleStr,
			}
		} else if p.RstpVersion &&
			p.Learn &&
			!p.Forward {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
				src: PtmMachineModuleStr,
			}
		}
	} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
		// events from Figure 17-22
		if p.RrWhileTimer.count == 0 &&
			!p.Sync &&
			!p.Learn &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}

		} else if !p.ReRoot &&
			!p.Sync &&
			!p.Learn &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndNotLearnSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}

		} else if p.RrWhileTimer.count == 0 &&
			!p.Sync &&
			p.Learn &&
			!p.Forward &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		} else if !p.ReRoot &&
			!p.Sync &&
			p.Learn &&
			!p.Forward &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		}
	}
}

func (p *StpPort) NotifyHelloWhenTimerExpired() {

	if p.PtxmMachineFsm.Machine.Curr.CurrentState() == PtxmStateIdle {
		if p.Selected &&
			!p.UpdtInfo {
			p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
				e:   PtxmEventHelloWhenEqualsZeroAndSelectedAndNotUpdtInfo,
				src: PtxmMachineModuleStr,
			}
		}
	}
}

func (p *StpPort) NotifyMdelayWhileTimerExpired() {

	if p.PpmmMachineFsm.Machine.Curr.CurrentState() == PpmmStateCheckingRSTP ||
		p.PpmmMachineFsm.Machine.Curr.CurrentState() == PpmmStateSelectingSTP {
		p.PpmmMachineFsm.PpmmEvents <- MachineEvent{
			e:   PpmmEventMdelayWhileEqualZero,
			src: PpmmMachineModuleStr,
		}
	}
}

func (p *StpPort) NotifyRbWhileTimerExpired() {

	if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateRootPort {
		if p.ReRoot &&
			p.RstpVersion &&
			!p.Learn &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventReRootedAndRbWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		} else if p.ReRoot &&
			p.RstpVersion &&
			p.Learn &&
			!p.Forward &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventReRootedAndRbWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		}
	}
}

func (p *StpPort) NotifyRcvdInfoWhileTimerExpired() {
	if p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateCurrent {
		if p.InfoIs == PortInfoStateReceived &&
			!p.UpdtInfo &&
			!p.RcvdMsg {
			p.PimMachineFsm.PimEvents <- MachineEvent{
				e:   PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg,
				src: PimMachineModuleStr,
			}
		}
	}
}

func (p *StpPort) NotifyRrWhileTimerExpired() {
	if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
		if p.ReRoot &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventRrWhileEqualZeroAndReRootAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		} else if p.FdWhileTimer.count == 0 &&
			!p.Sync &&
			!p.Learn &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}

		} else if p.Agreed &&
			!p.Sync &&
			!p.Learn &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		} else if p.OperEdge &&
			!p.Sync &&
			!p.Learn &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		} else if p.FdWhileTimer.count == 0 &&
			!p.Sync &&
			p.Learn &&
			!p.Forward &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		} else if p.Agreed &&
			!p.Sync &&
			p.Learn &&
			!p.Forward &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		} else if p.OperEdge &&
			!p.Sync &&
			p.Learn &&
			!p.Forward &&
			p.Selected &&
			!p.UpdtInfo {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
				src: PrtMachineModuleStr,
			}
		}
	}

}
func (p *StpPort) NotifyTcWhileTimerExpired() {

}
