// stm: #unit
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIInfo(t *testing.T) {
	apiInfo := defaultAPIInfo()
	// stm: @VENUSMINER_NODECONFIGAPIINFO_DIAL_ARGS_001
	url, err := apiInfo.DialArgs("0")
	assert.NoError(t, err)
	assert.Greater(t, len(url), 0)

	// stm: @VENUSMINER_NODECONFIGAPIINFO_HOST_001
	host, err := apiInfo.Host()
	assert.NoError(t, err)
	assert.Equal(t, host, "0.0.0.0:12308")

	// stm: @VENUSMINER_NODECONFIGAPIINFO_AUTH_HEADER_001
	apiInfo.Token = "hello"
	header := apiInfo.AuthHeader()
	assert.Equal(t, header.Get("Authorization"), "Bearer "+apiInfo.Token)
}
