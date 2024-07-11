// stm: #unit
package fsjournal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ipfs-force-community/sophon-miner/lib/journal"
	"github.com/ipfs-force-community/sophon-miner/node/repo"
	"github.com/stretchr/testify/require"
)

func TestFsJournal(t *testing.T) {
	tmp := t.TempDir()
	repo, err := repo.NewFS(tmp)
	require.NoError(t, err)
	lr, err := repo.Lock()
	require.NoError(t, err)

	dir := filepath.Join(lr.Path(), "journal")
	require.NoError(t, os.WriteFile(dir, []byte("file exists\n"), 0644))

	// stm: @VENUSMINER_JOURNAL_ENV_DISABLED_EVENTS_001
	envDisableEvent := journal.EnvDisabledEvents()
	require.Zero(t, len(envDisableEvent))

	{
		// 'tmpdir/journal' is a file would cause an error
		// stm: @VENUSMINER_FSJOURNAL_OPEN_FS_JOURNAL_002
		_, err = OpenFSJournal(lr, envDisableEvent)
		require.Error(t, err)
		require.NoError(t, os.RemoveAll(dir))
	}

	jlFile := filepath.Join(dir, "sophon-miner-journal.ndjson")
	{
		// If there is an error on rollJournalFile return nil and error.
		// stm: @VENUSMINER_FSJOURNAL_OPEN_FS_JOURNAL_003
		require.NoError(t, os.MkdirAll(jlFile, 0755))

		_, err := OpenFSJournal(lr, envDisableEvent)
		require.Error(t, err)
		require.NoError(t, os.RemoveAll(dir))
	}
}
