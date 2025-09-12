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

func getEnvFilePath() string {
	// The .env file has to be in the same folder as the neo4j-mcp binary
	// Get the filepath to the neo4j-mcp binary
	binaryFullPath, _ := filepath.Abs(os.Args[0])

	// filepath.Abs returns full folder path and the binary filename but we only want the full folder path so we
	// use filepath.Split that returns two values, dir = full folder path  and
	// file = the binary name
	// We just want the path which is where the .env file ( hopefully ) will be
	envFolderPath, _ := filepath.Split(binaryFullPath)

	return envFolderPath
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {

	// try loading .env and set the needed
	// environmental variables
	envFilePath := getEnvFilePath()
	envFileName := ".env"

	err := godotenv.Load(envFilePath + envFileName)

	// We didn't load the .env file :(
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not find .env file in %s\n", envFilePath)
		fmt.Fprintf(os.Stderr, "Trying Environment variables\n")
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
