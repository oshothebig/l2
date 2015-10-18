// timers
package lacp

import ()

// WaitWhileTimerStart
// Start the timer
func (p *LaAggPort) WaitWhileTimerStart() {
	p.waitWhileTimer.Reset(p.waitWhileTimerTimeout)
}

// WaitWhileTimerStop
// Stop the timer, which should only happen
// on creation as well as when the lacp mode is "on"
func (p *LaAggPort) WaitWhileTimerStop() {
	p.waitWhileTimer.Stop()
}

func (p *LaAggPort) CurrentWhileTimerStart() {
	p.currentWhileTimer.Reset(p.currentWhileTimerTimeout)
}

func (p *LaAggPort) CurrentWhileTimerStop() {
	p.currentWhileTimer.Stop()
}
