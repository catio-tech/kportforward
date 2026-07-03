package config

import (
	_ "embed"
)

// Embed the default configuration file
//
//go:embed default.yaml
var DefaultConfigYAML []byte

// Embed the default multi-environment configuration file (used with --multi-env)
//
//go:embed environments.yaml
var DefaultEnvironmentsYAML []byte
