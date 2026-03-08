package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
}

type TaskRepository interface {
	Create(ctx context.Context, t Task) (Task, error)
	List(ctx context.Context) ([]Task, error)
	Get(ctx context.Context, id string) (Task, error)
	Update(ctx context.Context, id string, patch Task) (Task, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, title string) ([]Task, error)
}

type PostgresRepo struct {
	db  *sql.DB
	log *zap.Logger
}

func New(dsn string, log *zap.Logger) (*PostgresRepo, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}
	repo := &PostgresRepo{db: db, log: log}
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return repo, nil
}

func (r *PostgresRepo) migrate() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id          TEXT        PRIMARY KEY,
			title       TEXT        NOT NULL,
			description TEXT        NOT NULL DEFAULT '',
			done        BOOLEAN     NOT NULL DEFAULT FALSE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func (r *PostgresRepo) Create(ctx context.Context, t Task) (Task, error) {
	t.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	t.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (id, title, description, done, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.ID, t.Title, t.Description, t.Done, t.CreatedAt,
	)
	if err != nil {
		r.log.Error("repository: create task failed",
			zap.String("component", "repository"),
			zap.String("error", err.Error()),
		)
		return Task{}, fmt.Errorf("create task: %w", err)
	}
	return t, nil
}

func (r *PostgresRepo) List(ctx context.Context) ([]Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, description, done, created_at FROM tasks ORDER BY created_at DESC`,
	)
	if err != nil {
		r.log.Error("repository: list tasks failed",
			zap.String("component", "repository"),
			zap.String("error", err.Error()),
		)
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Done, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *PostgresRepo) Get(ctx context.Context, id string) (Task, error) {
	var t Task
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, description, done, created_at FROM tasks WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Done, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return Task{}, fmt.Errorf("task not found")
	}
	if err != nil {
		r.log.Error("repository: get task failed",
			zap.String("component", "repository"),
			zap.String("error", err.Error()),
		)
		return Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

func (r *PostgresRepo) Update(ctx context.Context, id string, patch Task) (Task, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET title = $1, description = $2, done = $3 WHERE id = $4`,
		patch.Title, patch.Description, patch.Done, id,
	)
	if err != nil {
		r.log.Error("repository: update task failed",
			zap.String("component", "repository"),
			zap.String("error", err.Error()),
		)
		return Task{}, fmt.Errorf("update task: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *PostgresRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		r.log.Error("repository: delete task failed",
			zap.String("component", "repository"),
			zap.String("error", err.Error()),
		)
		return fmt.Errorf("delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

func (r *PostgresRepo) Search(ctx context.Context, title string) ([]Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, description, done, created_at
		 FROM tasks
		 WHERE title ILIKE $1
		 ORDER BY created_at DESC`,
		"%"+title+"%",
	)
	if err != nil {
		r.log.Error("repository: search tasks failed",
			zap.String("component", "repository"),
			zap.String("error", err.Error()),
		)
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Done, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}
