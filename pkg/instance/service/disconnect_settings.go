package instance_service

import instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"

// A disconnect changes connection state, not subscriptions or producer settings.
func markDisconnected(instance *instance_model.Instance, update func(string, bool, string) error) error {
	const reason = "Disconnected by API"
	if err := update(instance.Id, false, reason); err != nil {
		return err
	}
	instance.Connected = false
	instance.DisconnectReason = reason
	return nil
}
