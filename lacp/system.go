// system
package lacp

// 6.4.5 Variables associated with the System
type LacpSystem struct {
	// MAC address component of the System Id
	actor_system [6]uint8
	// System Priority
	actor_system_priority int
}
