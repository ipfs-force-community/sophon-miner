package modules

import (
	"context"
	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/lib/journal"
	"github.com/filecoin-project/venus-miner/lib/journal/fsjournal"
	"github.com/filecoin-project/venus-miner/node/repo"
)

func OpenFilesystemJournal(lr repo.LockedRepo, lc fx.Lifecycle, disabled journal.DisabledEvents) (journal.Journal, error) {
	jrnl, err := fsjournal.OpenFSJournal(lr, disabled)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error { return jrnl.Close() },
	})

	return jrnl, err
}
