package util

import (
	"database/sql"

	"github.com/google/uuid"
)

// ToNullUUID safely converts a *uuid.UUID to uuid.NullUUID
func ToNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

// ToNullInt32 safely converts an *int32 to sql.NullInt32
func ToNullInt32(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: *i, Valid: true}
}
