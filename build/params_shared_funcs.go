package build

import (
	"github.com/filecoin-project/go-address"
)

func SetAddressNetwork(n address.Network) {
	address.CurrentNetwork = n
}

func MustParseAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
