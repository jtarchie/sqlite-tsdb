//go:build cgo
// +build cgo

package main

import (
	_ "github.com/mattn/go-sqlite3"
)

const driverName = "sqlite3"
