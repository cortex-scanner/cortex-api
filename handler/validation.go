// Package handler provides validation utilities for HTTP handlers.
//
// # Request Body Validation
//
// ValidateRequestBody parses JSON from http.Request and validates struct fields.
// Field names are automatically derived from struct json tags using reflection:
//
//	type LoginRequest struct {
//	    Username string `json:"username"`
//	    Password string `json:"password"`
//	}
//
//	var req LoginRequest
//	err := ValidateRequestBody(r, &req,
//	    Field(&req.Username, Required(), Length(3, 20)),
//	    Field(&req.Password, Required(), Length(8, AnyLength)),
//	)
//
// # Struct Validation
//
// ValidateStruct validates struct fields using a fluent API similar to ozzo-validation:
//
//	type LoginRequest struct {
//	    Username string
//	    Password string
//	}
//
//	req := LoginRequest{Username: "john", Password: "secret123"}
//	err := ValidateStruct(
//	    Field("Username", req.Username, Required(), Length(3, 20)),
//	    Field("Password", req.Password, Required(), Length(8, AnyLength)),
//	)
//
// # Single Value Validation
//
// ValidateString validates individual string values:
//
//	id, err := ValidateString(r.PathValue("id"), UUID()).Validate()
//
// # Available Rules
//
// - Required(): validates non-empty values
// - Length(min, max): validates string length (use AnyLength for no limit)
// - Regex(pattern): validates against regex pattern
// - UUID(): validates UUID format
// - In(values...): validates value is in allowed list
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

// ValidationError represents a validation error for a single field or value
type ValidationError struct {
	Message string
}

// NewValidationError creates a new ValidationError
func NewValidationError(message string) ValidationError {
	return ValidationError{Message: message}
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
}

// StructValidationError represents validation errors for multiple struct fields
type StructValidationError struct {
	Errors map[string]error
}

// NewStructValidationError creates a new StructValidationError
func NewStructValidationError(errors map[string]error) StructValidationError {
	return StructValidationError{Errors: errors}
}

func (e StructValidationError) Error() string {
	var messages []string
	for field, err := range e.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", field, err.Error()))
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(messages, "; "))
}

type ValidationRule func(any) error

const AnyLength int = 0

func Length(min int, max int) ValidationRule {
	return func(value any) error {
		valueStr := value.(string)
		if min != AnyLength {
			if len(valueStr) < min {
				return NewValidationError(fmt.Sprintf("must be at least %d characters long", min))
			}
		}
		if max != AnyLength {
			if len(valueStr) > max {
				return NewValidationError(fmt.Sprintf("must be at most %d characters long", max))
			}
		}
		return nil
	}
}

func Regex(regex string) ValidationRule {
	regexCompiled := regexp.MustCompile(regex)
	return func(value any) error {
		valueStr := value.(string)
		if !regexCompiled.MatchString(valueStr) {
			return NewValidationError(fmt.Sprintf("must match regex %s", regex))
		}
		return nil
	}
}

func UUID() ValidationRule {
	return Regex("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
}

// Required validates that a value is not empty
func Required() ValidationRule {
	return func(value any) error {
		switch v := value.(type) {
		case string:
			if v == "" {
				return NewValidationError("is required")
			}
		case nil:
			return NewValidationError("is required")
		default:
			// For other types, just check for nil
			if v == nil {
				return NewValidationError("is required")
			}
		}
		return nil
	}
}

// In validates that a value is in a list of allowed values
func In(allowed ...string) ValidationRule {
	return func(value any) error {
		valueStr := value.(string)
		for _, v := range allowed {
			if valueStr == v {
				return nil
			}
		}
		return NewValidationError(fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
	}
}

type ValidationContext[T any] struct {
	fieldValueRaw any
	rules         []ValidationRule
}

func (ctx ValidationContext[T]) Validate() (T, error) {
	for _, rule := range ctx.rules {
		if err := rule(ctx.fieldValueRaw); err != nil {
			return ctx.fieldValueRaw.(T), err
		}
	}
	return ctx.fieldValueRaw.(T), nil
}

func ValidateString(value string, rules ...ValidationRule) *ValidationContext[string] {
	return &ValidationContext[string]{fieldValueRaw: value, rules: rules}
}

// FieldRules represents validation rules for a struct field
type FieldRules struct {
	FieldName string
	Value     any
	Rules     []ValidationRule
}

// FieldValidation represents a field pointer and its validation rules
type FieldValidation struct {
	FieldPtr any
	Rules    []ValidationRule
}

// Field creates a FieldValidation for use with ValidateRequestBody
// The fieldPtr must be a pointer to a field in the struct being validated
func Field(fieldPtr any, rules ...ValidationRule) FieldValidation {
	return FieldValidation{
		FieldPtr: fieldPtr,
		Rules:    rules,
	}
}

// fieldRulesCompat creates a FieldRules mapping for struct validation (backward compatibility)
func fieldRulesCompat(fieldName string, value any, rules ...ValidationRule) FieldRules {
	return FieldRules{
		FieldName: fieldName,
		Value:     value,
		Rules:     rules,
	}
}

// ValidateStruct validates a struct using field-rule mappings
func ValidateStruct(fields ...FieldRules) error {
	errors := make(map[string]error)

	for _, field := range fields {
		for _, rule := range field.Rules {
			if err := rule(field.Value); err != nil {
				errors[field.FieldName] = err
				break // Stop at first error for this field
			}
		}
	}

	if len(errors) > 0 {
		return NewStructValidationError(errors)
	}

	return nil
}

// getJSONFieldName uses reflection to find the JSON tag name for a field pointer
func getJSONFieldName[T any](target *T, fieldPtr any) (string, error) {
	targetValue := reflect.ValueOf(target).Elem()
	targetType := targetValue.Type()

	fieldPtrValue := reflect.ValueOf(fieldPtr)
	if fieldPtrValue.Kind() != reflect.Ptr {
		return "", NewValidationError("field must be a pointer")
	}

	fieldAddr := fieldPtrValue.Pointer()

	// Iterate through struct fields to find which one matches the pointer
	for i := 0; i < targetType.NumField(); i++ {
		field := targetValue.Field(i)
		if !field.CanAddr() {
			continue
		}

		if field.Addr().Pointer() == fieldAddr {
			// Found the matching field, extract JSON tag
			structField := targetType.Field(i)
			jsonTag := structField.Tag.Get("json")
			if jsonTag == "" {
				// No json tag, use field name in lowercase
				return strings.ToLower(structField.Name), nil
			}

			// Parse the json tag (format: "fieldname,omitempty")
			parts := strings.Split(jsonTag, ",")
			return parts[0], nil
		}
	}

	return "", NewValidationError("field not found in struct")
}

// ValidateRequestBody parses JSON from http.Request body and validates the result struct.
// The target parameter must be a pointer to the struct where parsed values will be stored.
// Field names are automatically derived from JSON struct tags using reflection.
//
// Example:
//
//	type LoginRequest struct {
//	    Username string `json:"username"`
//	    Password string `json:"password"`
//	}
//
//	var req LoginRequest
//	err := ValidateRequestBody(r, &req,
//	    Field(&req.Username, Required(), Length(3, 20)),
//	    Field(&req.Password, Required(), Length(8, AnyLength)),
//	)
func ValidateRequestBody[T any](r *http.Request, target *T, fields ...FieldValidation) error {
	// Parse JSON from request body
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return NewValidationError("invalid JSON in request body")
	}

	// Convert FieldValidation to FieldRules using reflection
	var fieldRules []FieldRules
	errors := make(map[string]error)

	for _, field := range fields {
		fieldName, err := getJSONFieldName(target, field.FieldPtr)
		if err != nil {
			errors[fmt.Sprintf("%v", field.FieldPtr)] = err
			continue
		}

		// Get the value from the field pointer
		fieldValue := reflect.ValueOf(field.FieldPtr).Elem().Interface()

		fieldRules = append(fieldRules, FieldRules{
			FieldName: fieldName,
			Value:     fieldValue,
			Rules:     field.Rules,
		})
	}

	// Check for errors during field name extraction
	if len(errors) > 0 {
		return NewStructValidationError(errors)
	}

	// Validate using existing ValidateStruct
	return ValidateStruct(fieldRules...)
}
