//go:build tools

package tools

import (
	// Linters
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/segmentio/golines"

	// Testing
	_ "github.com/stretchr/testify/mock"
)
