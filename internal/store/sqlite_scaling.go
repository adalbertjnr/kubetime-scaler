package store

import (
	"context"
	"database/sql"
)

type SqliteScalingOperationStore struct {
	db *sql.DB
}

func NewSqliteScalingOperationStore(db *sql.DB) *SqliteScalingOperationStore {
	return &SqliteScalingOperationStore{db: db}
}

func (so *SqliteScalingOperationStore) Get(ctx context.Context, scalingObject *ScalingOperation) error {
	query := `
		select
		 id, rule_name_description, resource_type,
		 replicas, created_at, updated_at
		 from scaling_operations
		 where resource_name = ? and namespace_name = ?;
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

func (so *SqliteScalingOperationStore) Bootstrap(ctx context.Context) error {
	query := `
		create table if not exists scaling_operations (
			id integer primary key autoincrement,
			namespace_name varchar(50) not null,
			rule_name_description text,
			resource_name varchar(50) not null,
			resource_type varchar(50),
			replicas integer not null,
			created_at datetime default current_timestamp,
			updated_at datetime default current_timestamp
		);
	`

	_, err := so.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	return nil
}

func (so *SqliteScalingOperationStore) Insert(ctx context.Context, scalingObject *ScalingOperation) error {
	query := `
		insert into scaling_operations
		(namespace_name, rule_name_description, resource_name, resource_type, replicas)
		values (?, ?, ?, ?, ?)
		returning id, created_at;
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

func (so *SqliteScalingOperationStore) Update(ctx context.Context, scalingObject *ScalingOperation) error {
	query := `
		update scaling_operations
		set replicas = ?, rule_name_description = ?, updated_at = current_timestamp
		where namespace_name = ? and resource_name = ? and resource_type = ?
		returning id, updated_at;
	`

	return so.db.QueryRowContext(
		ctx,
		query,
		scalingObject.Replicas,
		scalingObject.RuleNameDescription,
		scalingObject.NamespaceName,
		scalingObject.ResourceName,
		scalingObject.ResourceType,
	).Scan(
		&scalingObject.ID,
		&scalingObject.UpdatedAt,
	)
}
