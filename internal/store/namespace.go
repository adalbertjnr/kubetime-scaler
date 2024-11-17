package store

import "database/sql"

type NamespaceStorer interface {
	Create(*Namespace) error
	Update(*Namespace) error

	Get(string) (*Namespace, error)
	Delete(string) error
}

type NamespaceStore struct {
	db *sql.DB
}

type Namespace struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	DeploymentID int    `json:"deploymentID"`
}

func NewNamespaceStore(db *sql.DB) *NamespaceStore { return &NamespaceStore{db: db} }

func (d *NamespaceStore) Create(namespace *Namespace) error {
	return nil
}

func (d *NamespaceStore) Get(namespace string) (*Namespace, error) {
	return nil, nil
}

func (d *NamespaceStore) Update(namespace *Namespace) error {
	return nil
}

func (d *NamespaceStore) Delete(namespace string) error {
	return nil
}
