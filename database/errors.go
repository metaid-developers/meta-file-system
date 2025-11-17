package database

import "errors"

var (
	// ErrNotFound record not found
	ErrNotFound = errors.New("record not found")

	// ErrUnsupportedDBType unsupported database type
	ErrUnsupportedDBType = errors.New("unsupported database type")

	// ErrDatabaseClosed database is closed
	ErrDatabaseClosed = errors.New("database is closed")
)
