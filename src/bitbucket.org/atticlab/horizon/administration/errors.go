package administration

import (
    "fmt"
)

// AccountNotFoundError respresents error which occurred because requested 
// account wasn't found
type AccountNotFoundError struct {
    Address string
}

func (err AccountNotFoundError) Error() string {
    return fmt.Sprintf("Account with address %s wasn't found.", err.Address)
}

// InvalidFieldsError contains array if errors, corresponding to request fields
type InvalidFieldsError struct {
	Errors map[string]error
}

// NewInvalidFieldsError creates instance of InvalidFieldsError
func NewInvalidFieldsError() *InvalidFieldsError {
    return &InvalidFieldsError { Errors: make(map[string]error) }
}

func (err InvalidFieldsError) Error() string {
    errors := "Errors: \n"
    for key, value := range err.Errors {
        errors = errors + fmt.Sprintf("%s : %s", key, value)
    }
    
	return errors
}
