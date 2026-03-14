package settings

import "os"

const (
	EnvDev     = "dev"
	EnvStaging = "staging"
	EnvProd    = "prod"
)

// ResolveEnvironment determines the current environment.
// Priority: KVELMO_ENVIRONMENT env var > Settings.Environment > "dev" default.
func ResolveEnvironment(s *Settings) string {
	if env := os.Getenv("KVELMO_ENVIRONMENT"); env != "" {
		return env
	}
	if s != nil && s.Environment != "" {
		return s.Environment
	}

	return EnvDev
}
