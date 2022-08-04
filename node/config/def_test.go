package config

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"
)

func TestDefaultMinerRoundtrip(t *testing.T) {
	c := DefaultMinerConfig()

	var s string
	{
		buf := new(bytes.Buffer)
		_, _ = buf.WriteString("# Default config:\n")
		e := toml.NewEncoder(buf)
		require.NoError(t, e.Encode(c))

		s = buf.String()
	}

	c2, err := FromReader(strings.NewReader(s), DefaultMinerConfig())
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(c, c2))
}
