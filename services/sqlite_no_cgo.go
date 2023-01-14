//go:build !cgo
// +build !cgo

package services

import (
	_ "modernc.org/sqlite"
)

const dbDriverName = "sqlite"
