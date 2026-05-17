package client

// ServerConfig describes how to launch a language server.
type ServerConfig struct {
	ID         string
	Command    string
	Args       []string
	Extensions []string
	Env        map[string]string
	Disabled   bool
}
