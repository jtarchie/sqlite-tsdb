//go:build !cgo
// +build !cgo

package main

import (
	_ "modernc.org/sqlite"
)

const driverName = "sqlite"
