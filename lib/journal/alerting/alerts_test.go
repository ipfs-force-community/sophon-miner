// stm: #unit
package alerting

import (
	"encoding/json"
	"testing"

	mock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ipfs-force-community/sophon-miner/lib/journal"
	"github.com/ipfs-force-community/sophon-miner/lib/journal/mockjournal"
)

func TestAlerting(t *testing.T) {
	mockCtrl := mock.NewController(t)
	defer mockCtrl.Finish()
	j := mockjournal.NewMockJournal(mockCtrl)

	a := NewAlertingSystem(j)

	j.EXPECT().RegisterEventType("s1", "b1").Return(journal.EventType{System: "s1", Event: "b1"})
	// stm: @VENUSMINER_ALERTING_ADD_ALERT_TYPE_001
	al1 := a.AddAlertType("s1", "b1")

	j.EXPECT().RegisterEventType("s2", "b2").Return(journal.EventType{System: "s2", Event: "b2"})
	// stm: @VENUSMINER_ALERTING_ADD_ALERT_TYPE_001
	al2 := a.AddAlertType("s2", "b2")

	l := a.GetAlerts()
	require.Len(t, l, 2)
	require.Equal(t, al1, l[0].Type)
	require.Equal(t, al2, l[1].Type)

	for _, alert := range l {
		require.False(t, alert.Active)
		require.Nil(t, alert.LastActive)
		require.Nil(t, alert.LastResolved)
	}

	j.EXPECT().RecordEvent(a.alerts[al1].journalType, mock.Any())
	a.Raise(al1, "test")

	for _, alert := range l { // check for no magic mutations
		require.False(t, alert.Active)
		require.Nil(t, alert.LastActive)
		require.Nil(t, alert.LastResolved)
	}

	// stm: @VENUSMINER_ALERTING_GET_ALERTS_001
	l = a.GetAlerts()
	require.Len(t, l, 2)
	require.Equal(t, al1, l[0].Type)
	require.Equal(t, al2, l[1].Type)

	require.True(t, l[0].Active)
	require.NotNil(t, l[0].LastActive)
	require.Equal(t, "raised", l[0].LastActive.Type)
	require.Equal(t, json.RawMessage(`"test"`), l[0].LastActive.Message)
	require.Nil(t, l[0].LastResolved)

	require.False(t, l[1].Active)
	require.Nil(t, l[1].LastActive)
	require.Nil(t, l[1].LastResolved)

	var lastResolved *AlertEvent

	resolveMesage := "test resolve"
	j.EXPECT().RecordEvent(a.alerts[al1].journalType, mock.Any()).Do(
		func(arg0 interface{}, cb func() interface{}) {
			lastResolved = cb().(*AlertEvent)
		})
	// stm: @VENUSMINER_ALERTING_RAISE_001
	a.Resolve(al1, resolveMesage)
	l = a.GetAlerts()

	var resolvedAlert *Alert
	for _, alert := range l {
		if alert.Type.System == "s1" && alert.Type.Subsystem == "b1" {
			resolvedAlert = &alert
			break
		}
	}
	require.NotNil(t, resolvedAlert)
	require.NotNil(t, resolvedAlert.LastResolved)
	require.False(t, resolvedAlert.Active)
	require.Equal(t, resolvedAlert.LastResolved, lastResolved)
}
