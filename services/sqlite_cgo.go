//go:build cgo
// +build cgo

package services

import (
	_ "github.com/mattn/go-sqlite3"
)

const dbDriverName = "sqlite3"
