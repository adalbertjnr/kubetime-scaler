package store

import (
	"context"
	"database/sql"
)

type ScaleHistoryStorer interface {
	Create(context.Context, *ScaleHistory) error
	Update(context.Context, *ScaleHistory) error
}

type ScaleHistoryStore struct {
	db *sql.DB
}

func NewScaleHistoryStore(db *sql.DB) *ScaleHistoryStore { return &ScaleHistoryStore{db: db} }

type ScaleHistory struct {
	ID                  int    `json:"id"`
	NamespaceID         int    `json:"namespaceID"`
	RuleNameDescription string `json:"ruleNameDescription"`
	ResouceName         string `json:"resourceName"`
	ResourceType        string `json:"resourceType"`
	Replicas            int    `json:"replicas"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}

func (hs *ScaleHistoryStore) Create(ctx context.Context, historyObject *ScaleHistory) error {
	query := `
		insert into scale_history (namespace_id, resource_name, rule_name_description, resource_type, replicas)
		values ($1, $2, $3, $4)
		returning id, created_at
		`

	return hs.db.QueryRowContext(
		ctx,
		query,
		historyObject.NamespaceID,
		historyObject.ResouceName,
		historyObject.RuleNameDescription,
		historyObject.ResourceType,
		historyObject.Replicas,
	).Scan(
		&historyObject.ID,
		&historyObject.CreatedAt,
	)

}

func (hs *ScaleHistoryStore) Update(ctx context.Context, historyObject *ScaleHistory) error {
	query := `
		update scale_history 
		set replicas = $2, updated_at = now()
		where namespace_id = $1
		returning updated_at
	`

	return hs.db.QueryRowContext(
		ctx,
		query,
		historyObject.NamespaceID,
		historyObject.Replicas,
	).Scan(
		&historyObject.UpdatedAt,
	)
}
