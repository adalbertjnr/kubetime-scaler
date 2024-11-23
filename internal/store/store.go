package store

import "github.com/adalbertjnr/downscalerk8s/internal/db"

type Persistence struct {
	ScalingOperation ScalingOperationStorer
}

func New(enableDatabase bool) *Persistence {
	if !enableDatabase {
		return nil
	}

	dbClient := db.MustCreateClient()

	persistenceStore := Persistence{
		ScalingOperation: NewScalingOperationStore(dbClient),
	}

	return &persistenceStore
}
