//go:build tools
// +build tools

package tools

import (
	// Linters
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/segmentio/golines"
	_ "mvdan.cc/gofumpt"

	// Testing
	_ "github.com/stretchr/testify/mock"
)
