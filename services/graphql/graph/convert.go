package graph

import (
	"pz1/services/graphql/graph/model"
	"pz1/shared/repository"
)

// Этот файл НЕ перезаписывается gqlgen.
// Содержит вспомогательные функции конвертации между
// доменными моделями репозитория и GraphQL моделями.

func repoTaskToModel(t repository.Task) model.Task {
	return model.Task{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Done:        t.Done,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func repoTasksToModel(tasks []repository.Task) []*model.Task {
	result := make([]*model.Task, len(tasks))
	for i, t := range tasks {
		m := repoTaskToModel(t)
		result[i] = &m
	}
	return result
}
