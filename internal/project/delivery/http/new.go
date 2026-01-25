package http

import (
	"smap-project/internal/project"
	"smap-project/pkg/discord"
	pkgLog "smap-project/pkg/log"
)

type handler struct {
	l       pkgLog.Logger
	uc      project.UseCase
	discord *discord.Discord
}

func New(l pkgLog.Logger, uc project.UseCase, discord *discord.Discord) handler {
	return handler{
		l:       l,
		uc:      uc,
		discord: discord,
	}
}
