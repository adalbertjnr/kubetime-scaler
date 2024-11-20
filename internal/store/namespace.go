package store

import (
	"context"
	"database/sql"
)

type NamespaceStorer interface {
	InitDatabase(ctx context.Context) error
	Create(context.Context, *Namespace) error
}

type NamespaceStore struct {
	db *sql.DB
}

type Namespace struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

func NewNamespaceStore(db *sql.DB) *NamespaceStore { return &NamespaceStore{db: db} }

func (ns *NamespaceStore) InitDatabase(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS namespaces (
			id INTEGER PRIMARY KEY AUTOINCREMENT, 
			name TEXT NOT NULL, 
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS scale_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT, 
			namespace_id INTEGER NOT NULL, 
			rule_name_description TEXT, 
			resource_name TEXT, 
			resource_type TEXT, 
			replicas INTEGER NOT NULL, 
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
			FOREIGN KEY(namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE
		)`,
	}

	for i := range queries {
		_, err := ns.db.ExecContext(ctx, queries[i])
		return err
	}

	return nil
}

func (ns *NamespaceStore) Create(ctx context.Context, namespace *Namespace) error {
	query := `
		insert into namespaces (name)
		values ($1)
		returning id, created_at
		`

	return ns.db.QueryRowContext(
		ctx,
		query,
		namespace.Name,
	).Scan(
		&namespace.ID,
		&namespace.CreatedAt,
	)

}
