package instance_service

import (
	"errors"
	"testing"

	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
)

func TestDisconnectPreservesSubscriptionsAndProducers(t *testing.T) {
	instance := &instance_model.Instance{Id: "instance", Connected: true, Events: "MESSAGE,CALL", Webhook: "https://example.test/hook", RabbitmqEnable: "enabled"}
	calls := 0
	err := markDisconnected(instance, func(id string, connected bool, reason string) error {
		calls++
		if id != "instance" || connected || reason != "Disconnected by API" {
			t.Fatal("incorrect connection update")
		}
		return nil
	})
	if err != nil || calls != 1 || instance.Connected || instance.Events != "MESSAGE,CALL" || instance.Webhook != "https://example.test/hook" || instance.RabbitmqEnable != "enabled" {
		t.Fatalf("disconnect altered configuration: %#v, %v", instance, err)
	}
	failure := errors.New("database unavailable")
	instance.Connected = true
	instance.DisconnectReason = "previous reason"
	err = markDisconnected(instance, func(string, bool, string) error { return failure })
	if !errors.Is(err, failure) || !instance.Connected || instance.DisconnectReason != "previous reason" || instance.Events != "MESSAGE,CALL" {
		t.Fatalf("failed persistence published connection state: %#v, %v", instance, err)
	}
}
