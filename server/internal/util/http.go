package util

import "github.com/google/uuid"

func ParseOptionalUUID(param string) (*uuid.UUID, error) {
	if param == "" {
		return nil, nil
	}
	id, err := uuid.Parse(param)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
