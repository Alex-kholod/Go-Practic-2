package graph

import (
	"go.uber.org/zap"

	"pz1/shared/repository"
)

type Resolver struct {
	repo repository.TaskRepository
	log  *zap.Logger
}

func NewResolver(repo repository.TaskRepository, log *zap.Logger) *Resolver {
	return &Resolver{repo: repo, log: log}
}
