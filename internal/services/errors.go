package services

import "errors"

var (
	ErrEventNotFound             = errors.New("event not found")
	ErrRegistrationNotOpen       = errors.New("registration not open")
	ErrRegistrationDeadlinePassed = errors.New("registration deadline passed")
	ErrRegistrationFull          = errors.New("registration full")
)
