package service

import "errors"

var (
	ErrEmployeeValidate = errors.New("fullname and position are required")
	ErrInvalidID        = errors.New("ID should be positive number")
	ErrInvalidParams    = errors.New("invalid params")
	ErrNotFound         = errors.New("department not found")
	ErrCyclicDependency = errors.New("cyclic dependency detected: cannot move department inside its own subtree")
	ErrSelfParent       = errors.New("department cannot be its own parent")
)
