package whatsmeow_service

import (
	"sync"

	"go.mau.fi/whatsmeow/store"
)

var handshakeVersionMu sync.Mutex

// DeviceProps.Version describes the companion, not the version used by the
// handshake. SetWAVersion also updates BaseClientPayload in our pinned library.
func applyHandshakeVersion(version clientVersion) bool {
	if version.Major <= 0 || version.Minor <= 0 || version.Patch <= 0 {
		return false
	}
	handshakeVersionMu.Lock()
	defer handshakeVersionMu.Unlock()
	store.SetWAVersion(store.WAVersionContainer{uint32(version.Major), uint32(version.Minor), uint32(version.Patch)})
	return true
}
