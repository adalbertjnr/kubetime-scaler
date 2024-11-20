package store

import "github.com/adalbertjnr/downscalerk8s/internal/db"

type Persistence struct {
	ScaleHistory ScaleHistoryStorer
	Namespace    NamespaceStorer
}

func New(enableDatabase bool) *Persistence {
	if !enableDatabase {
		return nil
	}

	dbClient := db.MustCreateClient()

	persistenceStore := Persistence{
		ScaleHistory: NewScaleHistoryStore(dbClient),
		Namespace:    NewNamespaceStore(dbClient),
	}

	return &persistenceStore
}
