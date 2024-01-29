//go:build tools
// +build tools

package tools

import (
	// Linters
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "mvdan.cc/gofumpt"
	_ "github.com/segmentio/golines"

	// Test
	_ "github.com/stretchr/testify/mock"
)
