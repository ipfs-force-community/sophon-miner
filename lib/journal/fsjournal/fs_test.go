// stm: #unit
package fsjournal

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/venus-miner/lib/journal"
	"github.com/filecoin-project/venus-miner/node/repo"
	"github.com/stretchr/testify/require"
)

func TestFsJournal(t *testing.T) {
	tmp := t.TempDir()
	repo, err := repo.NewFS(tmp)
	require.NoError(t, err)
	lr, err := repo.Lock()
	require.NoError(t, err)

	dir := filepath.Join(lr.Path(), "journal")
	require.NoError(t, ioutil.WriteFile(dir, []byte("file exists\n"), 0644))

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

	jlFile := filepath.Join(dir, "venus-miner-journal.ndjson")
	{
		// If there is an error on rollJournalFile return nil and error.
		// stm: @VENUSMINER_FSJOURNAL_OPEN_FS_JOURNAL_003
		require.NoError(t, os.MkdirAll(jlFile, 0755))

		_, err := OpenFSJournal(lr, envDisableEvent)
		require.Error(t, err)
		require.NoError(t, os.RemoveAll(dir))
	}

	{
		// stm: @VENUSMINER_FSJOURNAL_OPEN_FS_JOURNAL_001
		jl, err := OpenFSJournal(lr, envDisableEvent)
		require.NoError(t, err)

		eType := jl.RegisterEventType("s1", "b1")
		require.NoError(t, err)

		isRecoded := make(chan bool, 1)
		// stm: @VENUSMINER_FSJOURNAL_RECORD_EVENT_001
		jl.RecordEvent(eType, func() interface{} {
			isRecoded <- true
			return "hello"
		})
		t.Logf("wating `RecodeEvent` execution...")
		<-isRecoded

		require.NoError(t, jl.Close())

		recordEventData, err := ioutil.ReadFile(jlFile)
		require.NoError(t, err)

		event := &journal.Event{}
		require.NoErrorf(t, json.Unmarshal(recordEventData, event),
			"json unmarshal: [%s] to journal.Event failed.", string(recordEventData))
		if message, isok := event.Data.(string); !isok {
			t.Errorf("event.Data should be a string")
		} else {
			require.Equal(t, message, "hello")
		}
	}
}
