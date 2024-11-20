package store

import (
	"context"
	"database/sql"
)

type NamespaceStorer interface {
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
