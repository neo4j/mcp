package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	URI      string
	Username string
	Password string
	Database string
}

// Returns the full filepath to the neo4j-mcp binary or an error on failure.
func getEnvFilePath() (string, error) {
	// The .env file has to be in the same folder as the neo4j-mcp binary
	// Get the filepath to the neo4j-mcp binary

	var err error
	var binaryFullPath, envFolderPath string

	binaryFullPath, err = filepath.Abs(os.Args[0])
	if err != nil {
		return "", err
	}

	// filepath.Abs returns full folder path and the binary filename but we only want the full folder path so we
	// use filepath.Split that returns two values, dir = full folder path  and
	// file = the binary name
	// We just want the path which is where the .env file ( hopefully ) will be
	envFolderPath, _ = filepath.Split(binaryFullPath)

	return envFolderPath, nil
}

// Sets environment variables from .env file located in the same folder as the neo4j-mcp binary
func setEnvFromFile() error {
	var err error
	var envFilePath, envFileName string

	envFileName = ".env"

	// Get the filepath to the .env file
	envFilePath, err = getEnvFilePath()
	if err != nil {
		return err
	}

	err = godotenv.Load(filepath.Join(envFilePath, envFileName))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not find .env in %s\n", envFilePath)
		return err
	}

	fmt.Fprintf(os.Stderr, "Using values in .env to set environmental variables\n")

	return nil
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {

	err := setEnvFromFile()
	// We didn't load the .env file :(
	if err != nil {
		fmt.Fprintf(os.Stderr, "Using values from environment variables\n")
	}

	return &Config{
		URI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Username: getEnv("NEO4J_USERNAME", "neo4j"),
		Password: getEnv("NEO4J_PASSWORD", "password"),
		Database: getEnv("NEO4J_DATABASE", "neo4j"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
