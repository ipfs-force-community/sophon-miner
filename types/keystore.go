package types

import "github.com/filecoin-project/venus/venus-shared/types"

// KeyStore is used for storing secret keys
type KeyStore interface {
	// List lists all the keys stored in the KeyStore
	List() ([]string, error)
	// Get gets a key out of keystore and returns KeyInfo corresponding to named key
	Get(string) (types.KeyInfo, error)
	// Put saves a key info under given name
	Put(string, types.KeyInfo) error
	// Delete removes a key from keystore
	Delete(string) error
}
