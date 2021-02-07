package dtypes

import (
	"github.com/ipfs/go-datastore"
)

// MetadataDS stores metadata
// dy default it's namespaced under /metadata in main repo datastore
type MetadataDS datastore.Batching
