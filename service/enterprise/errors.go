package enterprise

import "errors"

var (
	ErrDepartmentNotFound           = errors.New("department not found")
	ErrDepartmentNameHistoryInvalid = errors.New("department name history invalid")
	ErrUserNotFound                 = errors.New("enterprise user not found")
	ErrMembershipNotFound           = errors.New("enterprise membership not found")
	ErrMembershipAlreadyExists      = errors.New("enterprise membership already exists")
	ErrDuplicateDepartment          = errors.New("duplicate department id")
	ErrInvalidMembershipInput       = errors.New("invalid enterprise membership input")
	ErrDepartmentAdminRequired      = errors.New("enterprise department admin permission required")
)
