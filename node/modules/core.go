package modules

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	types2 "github.com/filecoin-project/venus-miner/types"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/node/repo"
)

const (
	JWTSecretName   = "auth-jwt-private" //nolint:gosec
	KTJwtHmacSecret = "jwt-hmac-secret"  //nolint:gosec
)

var (
	log = logging.Logger("modules")
)

type JwtPayload struct {
	Allow []auth.Permission
}

func APISecret(keystore types2.KeyStore, lr repo.LockedRepo) (*types2.APIAlg, error) {
	key, err := keystore.Get(JWTSecretName)

	if errors.Is(err, types2.ErrKeyInfoNotFound) {
		log.Warn("Generating new API secret")

		sk, err := ioutil.ReadAll(io.LimitReader(rand.Reader, 32))
		if err != nil {
			return nil, err
		}

		key = types2.KeyInfo{
			Type:       KTJwtHmacSecret,
			PrivateKey: sk,
		}

		if err := keystore.Put(JWTSecretName, key); err != nil {
			return nil, fmt.Errorf("writing API secret: %w", err)
		}

		// TODO: make this configurable
		p := JwtPayload{
			Allow: api.AllPermissions,
		}

		cliToken, err := jwt.Sign(&p, jwt.NewHS256(key.PrivateKey))
		if err != nil {
			return nil, err
		}

		if err := lr.SetAPIToken(cliToken); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("could not get JWT Token: %w", err)
	}

	return (*types2.APIAlg)(jwt.NewHS256(key.PrivateKey)), nil
}
