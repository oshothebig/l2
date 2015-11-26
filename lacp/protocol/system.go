// system
package lacp

import (
	"net"
)

// 6.4.5 Variables associated with the System
type LacpSystem struct {
	// System Priority
	actor_system_priority uint16
	// MAC address component of the System Id
	actor_system net.HardwareAddr
}

func (s *LacpSystem) LacpSystemActorSystemIdSet(actor_system net.HardwareAddr) {
	s.actor_system = actor_system
}

func (s *LacpSystem) LacpSystemActorSystemPrioritySet(actor_system_priority uint16) {
	s.actor_system_priority = actor_system_priority
}
