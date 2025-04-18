package prompt

import (
	"fmt"
	"strings"
)

func NewMissingFieldsError(fields []string) *MissingFieldsError {
	return &MissingFieldsError{MissingFields: fields}
}

type MissingFieldsError struct {
	MissingFields []string `json:"missing_fields"`
}

func (e *MissingFieldsError) Error() string {
	var errMsg strings.Builder
	errMsg.WriteString("Missing: \n")
	for _, field := range e.MissingFields {
		errMsg.WriteString(fmt.Sprintf("%v, ", field))
	}
	return errMsg.String()
}

// func NewMissingFieldsForTemplatesError(fields map[string]MissingFieldsError) *MissingFieldsForTemplatesError {
// 	return &MissingFieldsForTemplatesError{
// 		MissingFields: fields,
// 	}
// }

// type MissingFieldsForTemplatesError struct {
// 	MissingFields map[string]MissingFieldsError `json:"missing_fields"`
// }

// func (e *MissingFieldsForTemplatesError) Error() string {
// 	var errMsg strings.Builder
// 	errMsg.WriteString("Missing required variables in templates:\n")

// 	for tmplPath, fields := range e.MissingFields {
// 		errMsg.WriteString(fmt.Sprintf("  Template %q is missing:\n", tmplPath))
// 		errMsg.WriteString(fields.Error())
// 		errMsg.WriteString("\n")
// 	}

// 	return errMsg.String()
// }
