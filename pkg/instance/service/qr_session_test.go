package instance_service

import (
	"errors"
	"testing"
)

type testQRSession struct{ loggedIn, connected bool }

func (s testQRSession) IsLoggedIn() bool  { return s.loggedIn }
func (s testQRSession) IsConnected() bool { return s.connected }

func TestQRPollingPreservesActiveAndPairingSessions(t *testing.T) {
	for _, test := range []struct {
		name     string
		client   qrSession
		starts   int
		loggedIn bool
	}{
		{"authenticated", testQRSession{true, true}, 0, true},
		{"authenticated but temporarily disconnected", testQRSession{true, false}, 0, true},
		{"pairing", testQRSession{false, true}, 0, false},
		{"missing", nil, 1, false},
		{"disconnected without session", testQRSession{}, 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			started, err := prepareQRSession(test.client, func() error { calls++; return nil })
			if calls != test.starts || started != (test.starts == 1) || errors.Is(err, errQRSessionLoggedIn) != test.loggedIn {
				t.Fatalf("unexpected QR lifecycle: starts=%d started=%v err=%v", calls, started, err)
			}
		})
	}
	failure := errors.New("storage unavailable")
	if started, err := prepareQRSession(nil, func() error { return failure }); started || !errors.Is(err, failure) {
		t.Fatalf("startup failure was not propagated: %v %v", started, err)
	}
}
