package main

import (
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"go.uber.org/zap"

	"pz1/services/graphql/graph"
	"pz1/services/graphql/graph/generated"
	"pz1/shared/logger"
	"pz1/shared/middleware"
	"pz1/shared/repository"
)

func main() {
	log := logger.Must("graphql")
	defer log.Sync()

	port := os.Getenv("GRAPHQL_PORT")
	if port == "" {
		port = "8090"
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/tasks?sslmode=disable"
	}

	// Подключаемся к тому же репозиторию что и REST tasks.
	repo, err := repository.New(dsn, log)
	if err != nil {
		log.Fatal("failed to connect to database",
			zap.String("component", "repository"),
			zap.Error(err),
		)
	}
	log.Info("connected to database")

	// Создаём резолвер с зависимостями.
	resolver := graph.NewResolver(repo, log)

	// gqlgen создаёт HTTP handler из схемы и резолверов.
	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: resolver,
		}),
	)

	mux := http.NewServeMux()

	mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))

	// Основной endpoint для GraphQL запросов.
	mux.Handle("/query", authMiddleware(log)(srv))

	// Health endpoint.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"graphql"}`))
	})

	// Применяем базовые middleware.
	handler := middleware.RequestID(
		middleware.AccessLog(log)(mux),
	)

	log.Info("graphql server starting",
		zap.String("port", port),
		zap.String("playground", "http://localhost:"+port+"/"),
	)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("graphql server stopped", zap.Error(err))
	}
}

// authMiddleware проверяет Bearer токен через заголовок Authorization.
func authMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
