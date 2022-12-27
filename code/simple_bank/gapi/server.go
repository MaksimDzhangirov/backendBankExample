package gapi

import (
	"fmt"

	"github.com/MaksimDzhangirov/backendBankExample/pb"
	"github.com/MaksimDzhangirov/backendBankExample/worker"

	db "github.com/MaksimDzhangirov/backendBankExample/db/sqlc"
	"github.com/MaksimDzhangirov/backendBankExample/token"
	"github.com/MaksimDzhangirov/backendBankExample/util"
)

// Server обслуживает gRPC запросы нашего банковского сервиса.
type Server struct {
	pb.UnimplementedSimpleBankServer
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
}

// NewServer создаёт новый HTTP сервер и настраивает маршрутизацию.
func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:          config,
		store:           store,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
	}

	return server, nil
}
