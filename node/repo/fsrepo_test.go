// stm: #unit
package repo

import (
	"os"
	"testing"

	"github.com/ipfs-force-community/sophon-miner/node/config"
	"github.com/stretchr/testify/assert"
)

func genFsRepo(t *testing.T) (*FsRepo, func()) {
	path, err := os.MkdirTemp("", "venus-repo-")
	if err != nil {
		t.Fatal(err)
	}

	repo, err := NewFS(path)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.Init()
	if err != ErrRepoExists && err != nil {
		t.Fatal(err)
	}
	return repo, func() {
		_ = os.RemoveAll(path)
	}
}

func TestFsBasic(t *testing.T) {
	// stm: @VENUSMINER_NODE_REPO_FSREPO_INIT_001
	repo, closer := genFsRepo(t)

	// stm: @VENUSMINER_NODE_REPO_FSREPO_EXISTS_001
	exists, err := repo.Exists()
	assert.NoError(t, err)
	assert.True(t, exists)

	cfg := config.DefaultMinerConfig()
	// stm: @VENUSMINER_NODE_REPO_FSREPO_UPDATE_001
	err = repo.Update(cfg)
	assert.NoError(t, err)

	defer closer()
	// stm: @VENUSMINER_NODE_REPO_FSREPO_API_ENDPOINT_001, @VENUSMINER_NODE_REPO_FSREPO_API_TOKEN_001, @VENUSMINER_NODE_REPO_FSREPO_API_CONFIG_001, @VENUSMINER_NODE_REPO_FSREPO_LOCK_001, @VENUSMINER_NODE_REPO_FSREPO_PATH_001, @VENUSMINER_NODE_REPO_FSREPO_CLOSE_001, @VENUSMINER_NODE_REPO_FSREPO_SET_CONFIG_001, @VENUSMINER_NODE_REPO_FSREPO_SET_API_ENDPOINT_001, @VENUSMINER_NODE_REPO_FSREPO_SET_API_TOKEN_001, @VENUSMINER_NODE_REPO_FSREPO_LIST_001, @VENUSMINER_NODE_REPO_FSREPO_GET_001, @VENUSMINER_NODE_REPO_FSREPO_PUT_001, @VENUSMINER_NODE_REPO_FSREPO_DELETE_001
	basicTest(t, repo)
}
