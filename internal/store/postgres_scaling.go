package store

import (
	"context"
	"database/sql"
)

type PostgresScalingOperationStore struct {
	db *sql.DB
}

func NewPostgresScalingOperationStore(db *sql.DB) *PostgresScalingOperationStore {
	return &PostgresScalingOperationStore{db: db}
}

func (so *PostgresScalingOperationStore) Get(ctx context.Context, scalingObject *ScalingOperation) error {
	query := `
		select
		 id, rule_name_description, resource_type,
		 replicas, created_at, updated_at
		 from scaling_operations
		 where resource_name = $1 and namespace_name = $2
	`

	return so.db.QueryRowContext(
		ctx,
		query,
		scalingObject.ResourceName,
		scalingObject.NamespaceName,
	).Scan(
		&scalingObject.ID,
		&scalingObject.RuleNameDescription,
		&scalingObject.ResourceType,
		&scalingObject.Replicas,
		&scalingObject.CreatedAt,
		&scalingObject.UpdatedAt,
	)
}

func (so *PostgresScalingOperationStore) Bootstrap(ctx context.Context) error {
	query := `
		create table if not exists scaling_operations (
			id serial primary key,
			namespace_name varchar(50) not null,
			rule_name_description text,
			resource_name varchar(50) not null,
			resource_type varchar(50),
			replicas integer not null,
			created_at timestamp default current_timestamp,
			updated_at timestamp default current_timestamp
		);
	`

	_, err := so.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	return nil
}

func (so *PostgresScalingOperationStore) Insert(ctx context.Context, scalingObject *ScalingOperation) error {
	query := `
		insert into scaling_operations
		(namespace_name, rule_name_description, resource_name, resource_type, replicas)
		values ($1, $2, $3, $4, $5)
		returning id, created_at
	`

	return so.db.QueryRowContext(
		ctx,
		query,
		scalingObject.NamespaceName,
		scalingObject.RuleNameDescription,
		scalingObject.ResourceName,
		scalingObject.ResourceType,
		scalingObject.Replicas,
	).Scan(
		&scalingObject.ID,
		&scalingObject.CreatedAt,
	)
}

func (so *PostgresScalingOperationStore) Update(ctx context.Context, scalingObject *ScalingOperation) error {
	query := `
		update scaling_operations
		set replicas = $2, resource_name = $3, rule_name_description = $4,
		resource_type = $5, updated_at = now()
		where namespace_name = $1 and resource_name = $3
		returning id, updated_at
	`

	return so.db.QueryRowContext(
		ctx,
		query,
		scalingObject.NamespaceName,
		scalingObject.Replicas,
		scalingObject.ResourceName,
		scalingObject.RuleNameDescription,
		scalingObject.ResourceType,
	).Scan(
		&scalingObject.ID,
		&scalingObject.UpdatedAt,
	)

}
