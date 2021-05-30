package config

type AuthConfig struct {
	ListenAPI string // http://[ip]:[port]
	Token     string
}

func newDefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		ListenAPI: "",
		Token:     "",
	}
}
