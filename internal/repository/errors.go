package repository

import "errors"

var (
	ErrNilEmployee   = errors.New("employee cannot be empty")
	ErrNilDepartment = errors.New("department cannot be empty")
	ErrNotFound      = errors.New("record not found")
	ErrHasChildren   = errors.New("record has children")
	ErrAlreadyExists = errors.New("record already exists")
	ErrCycle         = errors.New("department tree cycle")
)
