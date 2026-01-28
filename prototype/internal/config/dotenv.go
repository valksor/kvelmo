package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const (
	// MehrhofDir is the name of the mehrhof configuration directory.
	MehrhofDir = ".mehrhof"
	// EnvFileName is the name of the environment variables file.
	EnvFileName = ".env"
)

// LoadDotEnv loads environment variables from .mehrhof/.env if it exists.
// It uses godotenv.Overload() which overrides existing environment variables
// (.env values take priority over system env vars).
// Returns nil if the file doesn't exist (not an error condition).
// Returns error only if the file exists but cannot be parsed.
func LoadDotEnv(baseDir string) error {
	envPath := filepath.Join(baseDir, MehrhofDir, EnvFileName)

	// Check if file exists - silently skip if not
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil
	}

	// Load the .env file - override any existing shell env vars
	return godotenv.Overload(envPath)
}

// LoadDotEnvFromCwd loads .env from current working directory's .mehrhof/.env.
func LoadDotEnvFromCwd() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	return LoadDotEnv(cwd)
}
