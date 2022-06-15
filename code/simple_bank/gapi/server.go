package gapi

import (
	"fmt"
	"github.com/MaksimDzhangirov/backendBankExample/pb"

	db "github.com/MaksimDzhangirov/backendBankExample/db/sqlc"
	"github.com/MaksimDzhangirov/backendBankExample/token"
	"github.com/MaksimDzhangirov/backendBankExample/util"
)

// Server обслуживает gRPC запросы нашего банковского сервиса.
type Server struct {
	pb.UnimplementedSimpleBankServer
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}

// NewServer создаёт новый HTTP сервер и настраивает маршрутизацию.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	return server, nil
}