package http

import (
	"smap-project/internal/webhook"
	"smap-project/pkg/discord"
	pkgLog "smap-project/pkg/log"
)

type handler struct {
	l           pkgLog.Logger
	uc          webhook.UseCase
	discord     *discord.Discord
	internalKey string
}

func New(l pkgLog.Logger, uc webhook.UseCase, discord *discord.Discord, internalKey string) handler {
	return handler{
		l:           l,
		uc:          uc,
		discord:     discord,
		internalKey: internalKey,
	}
}
