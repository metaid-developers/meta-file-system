package database

import "errors"

var (
	ErrNotFound          = errors.New("record not found")
	ErrUnsupportedDBType = errors.New("unsupported database type")
	ErrNotImplemented    = errors.New("not implemented")
)
