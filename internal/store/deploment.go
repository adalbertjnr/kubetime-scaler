package store

import "database/sql"

type DeploymentStorer interface {
	Create(*Deployment) error
	Update(*Deployment) error

	Get(string) (*Deployment, error)
	Delete(string) error
}

type DeploymentStore struct {
	db *sql.DB
}

func NewDeploymentStore(db *sql.DB) *DeploymentStore { return &DeploymentStore{db: db} }

type Deployment struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Replicas int    `json:"replicas"`
}

func (d *DeploymentStore) Create(deployment *Deployment) error {
	return nil
}

func (d *DeploymentStore) Get(deployment string) (*Deployment, error) {
	return nil, nil
}

func (d *DeploymentStore) Update(deployment *Deployment) error {
	return nil
}

func (d *DeploymentStore) Delete(deployment string) error {
	return nil
}
