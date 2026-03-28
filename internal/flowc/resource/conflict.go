package resource

import (
	"errors"
	"fmt"
)

var (
	// ErrRevisionConflict indicates the caller supplied a stale revision.
	ErrRevisionConflict = errors.New("revision conflict: resource has been modified")

	// ErrOwnershipConflict indicates the resource is managed by a different writer
	// and the conflict policy is "strict".
	ErrOwnershipConflict = errors.New("ownership conflict: resource is managed by another writer")

	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates the resource already exists (used for create-only paths).
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidResource indicates the resource failed validation.
	ErrInvalidResource = errors.New("invalid resource")
)

// RevisionConflictError provides details about a revision mismatch.
type RevisionConflictError struct {
	Key      ResourceKey
	Expected int64
	Actual   int64
}

func (e *RevisionConflictError) Error() string {
	return fmt.Sprintf("revision conflict on %s: expected %d, actual %d", e.Key, e.Expected, e.Actual)
}

func (e *RevisionConflictError) Unwrap() error { return ErrRevisionConflict }

// OwnershipConflictError provides details about an ownership mismatch.
type OwnershipConflictError struct {
	Key           ResourceKey
	CurrentOwner  string
	AttemptedBy   string
}

func (e *OwnershipConflictError) Error() string {
	return fmt.Sprintf("ownership conflict on %s: owned by %q, attempted by %q", e.Key, e.CurrentOwner, e.AttemptedBy)
}

func (e *OwnershipConflictError) Unwrap() error { return ErrOwnershipConflict }
