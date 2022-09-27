package repo

import (
	"os"
	"testing"
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
	repo, closer := genFsRepo(t)
	defer closer()
	basicTest(t, repo)
}
