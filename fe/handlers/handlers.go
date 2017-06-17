package handlers

import (
	"pixur.org/pixur/fe/server"
)

var (
	regfuncs []server.RegFunc
)

func register(rf server.RegFunc) {
	regfuncs = append(regfuncs, rf)
}

func RegisterAll(s *server.Server) {
	for _, rf := range regfuncs {
		s.Register(rf)
	}
}
