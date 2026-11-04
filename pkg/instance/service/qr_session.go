package instance_service

import (
	"errors"
	"fmt"
)

var errQRSessionLoggedIn = errors.New("session already logged in")

type qrSession interface {
	IsLoggedIn() bool
	IsConnected() bool
}

// Polling for QR must never replace an authenticated session or interrupt a
// connected client waiting for pairing. Only an absent/disconnected client starts.
func prepareQRSession(client qrSession, start func() error) (bool, error) {
	if client != nil && client.IsLoggedIn() {
		return false, errQRSessionLoggedIn
	}
	if client == nil || !client.IsConnected() {
		if err := start(); err != nil {
			return false, fmt.Errorf("failed to start instance: %w", err)
		}
		return true, nil
	}
	return false, nil
}
