package whatsmeow_service

import (
	"testing"

	"go.mau.fi/whatsmeow/store"
)

func TestHandshakeUsesResolvedVersion(t *testing.T) {
	original := store.GetWAVersion()
	t.Cleanup(func() { store.SetWAVersion(original) })
	version := clientVersion{Major: 2, Minor: 3000, Patch: 123456789}
	if !applyHandshakeVersion(version) {
		t.Fatal("valid version was not applied")
	}
	if got := store.GetWAVersion(); got != (store.WAVersionContainer{2, 3000, 123456789}) {
		t.Fatalf("wrong handshake version: %v", got)
	}
	payload := store.BaseClientPayload.GetUserAgent().GetAppVersion()
	if payload.GetPrimary() != 2 || payload.GetSecondary() != 3000 || payload.GetTertiary() != 123456789 {
		t.Fatalf("client payload still uses a different version: %v", payload)
	}
	for _, invalid := range []clientVersion{{}, {Major: -1, Minor: 3000, Patch: 1}, {Major: 2, Minor: 0, Patch: 1}} {
		if applyHandshakeVersion(invalid) || store.GetWAVersion() != (store.WAVersionContainer{2, 3000, 123456789}) {
			t.Fatal("an unresolved version replaced the valid handshake version")
		}
	}
}
