// stm: #uint
package repo

import (
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"

	"github.com/filecoin-project/venus-miner/node/config"
)

func basicTest(t *testing.T, repo Repo) {
	apima, err := repo.APIEndpoint()
	if assert.Error(t, err) {
		assert.Equal(t, ErrNoAPIEndpoint, err)
	}
	assert.Nil(t, apima, "with no api endpoint, return should be nil")

	lrepo, err := repo.Lock()
	assert.NoError(t, err, "should be able to lock once")
	assert.NotNil(t, lrepo, "locked repo shouldn't be nil")

	{
		lrepo2, err := repo.Lock()
		if assert.Error(t, err) {
			assert.Equal(t, ErrRepoAlreadyLocked, err)
		}
		assert.Nil(t, lrepo2, "with locked repo errors, nil should be returned")
	}

	err = lrepo.Close()
	assert.NoError(t, err, "should be able to unlock")

	lrepo, err = repo.Lock()
	assert.NoError(t, err, "should be able to relock")
	assert.NotNil(t, lrepo, "locked repo shouldn't be nil")

	ma, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/43244")
	assert.NoError(t, err, "creating multiaddr shouldn't error")

	err = lrepo.SetAPIEndpoint(ma)
	assert.NoError(t, err, "setting multiaddr shouldn't error")

	apima, err = repo.APIEndpoint()
	assert.NoError(t, err, "setting multiaddr shouldn't error")
	assert.Equal(t, ma, apima, "returned API multiaddr should be the same")

	err = lrepo.SetAPIToken([]byte("api token"))
	assert.NoError(t, err)

	token, err := repo.APIToken()
	assert.NoError(t, err)
	assert.Equal(t, token, []byte("api token"))

	c1, err := lrepo.Config()
	assert.Equal(t, config.DefaultMinerConfig(), c1, "there should be a default config")
	assert.NoError(t, err, "config should not error")

	err = lrepo.Close()
	assert.NoError(t, err, "should be able to close")

	apima, err = repo.APIEndpoint()
	assert.NoError(t, err, "getting multiaddr shouldn't error")
	assert.Equal(t, ma, apima, "returned API multiaddr should be the same")

	lrepo, err = repo.Lock()
	assert.NoError(t, err, "should be able to relock")
	assert.NotNil(t, lrepo, "locked repo shouldn't be nil")
}
