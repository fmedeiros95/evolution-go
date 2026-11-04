package whatsmeow_service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/EvolutionAPI/evolution-go/pkg/config"
	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
	logger_wrapper "github.com/EvolutionAPI/evolution-go/pkg/logger"
	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type archiveWebhookRecorder struct {
	WhatsmeowService
	output chan []byte
}

func (r archiveWebhookRecorder) CallWebhook(_ *instance_model.Instance, _ string, data []byte) {
	r.output <- data
}

func TestArchiveEventReachesWebhookWithoutPanic(t *testing.T) {
	cfg := &config.Config{LogDirectory: t.TempDir()}
	recorder := archiveWebhookRecorder{output: make(chan []byte, 1)}
	client := &MyClient{
		service: recorder, userID: "archive-test", token: "test-token",
		Instance: &instance_model.Instance{Name: "archive-test"},
		config:   cfg, loggerWrapper: logger_wrapper.NewLoggerManager(cfg),
	}
	t.Cleanup(func() { _ = client.loggerWrapper.GetLogger("archive-test").Close() })
	client.myEventHandler(&events.Archive{
		JID:       types.NewJID("5511999999999", types.DefaultUserServer),
		Timestamp: time.Unix(1700000000, 0), FromFullSync: true,
		Action: &waSyncAction.ArchiveChatAction{Archived: proto.Bool(true)},
	})
	select {
	case raw := <-recorder.output:
		var payload struct {
			Event string `json:"event"`
			Data  struct {
				JID          string
				Timestamp    time.Time
				FromFullSync bool
				Action       struct {
					Archived bool `json:"archived"`
				}
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Event != "Archive" || payload.Data.JID != "5511999999999@s.whatsapp.net" || !payload.Data.FromFullSync || !payload.Data.Action.Archived || payload.Data.Timestamp.Unix() != 1700000000 {
			t.Fatalf("archive event changed during dispatch: %s", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Archive webhook was not dispatched")
	}
}
