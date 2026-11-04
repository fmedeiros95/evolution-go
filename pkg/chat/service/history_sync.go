package chat_service

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

type historySyncSender interface {
	BuildHistorySyncRequest(lastKnownMessageInfo *types.MessageInfo, count int) *waE2E.Message
	SendPeerMessage(ctx context.Context, message *waE2E.Message) (whatsmeow.SendResponse, error)
}

func requestHistorySync(ctx context.Context, sender historySyncSender, info *types.MessageInfo, count int) (whatsmeow.SendResponse, error) {
	return sender.SendPeerMessage(ctx, sender.BuildHistorySyncRequest(info, count))
}
