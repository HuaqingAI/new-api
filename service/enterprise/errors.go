package enterprise

import "errors"

var (
	ErrDepartmentNotFound           = errors.New("department not found")
	ErrDepartmentNameHistoryInvalid = errors.New("department name history invalid")
)
