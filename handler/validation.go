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
// String rules:
// - Required(): validates non-empty values
// - Length(min, max): validates string length (use AnyLength for no limit)
// - Regex(pattern): validates against regex pattern
// - UUID(): validates UUID format
// - In(values...): validates value is in allowed list
//
// Numeric rules:
// - Min(min): validates minimum value for int, int64, float64
// - Max(max): validates maximum value for int, int64, float64
// - Range(min, max): validates value is within range
//
// Slice/Array rules:
// - MinItems(min): validates minimum number of elements
// - MaxItems(max): validates maximum number of elements
// - Each(rules...): validates each element in slice
//
// Map rules:
// - MinItems(min): validates minimum number of entries
// - MaxItems(max): validates maximum number of entries
// - Keys(rules...): validates each key in map
// - Values(rules...): validates each value in map
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

// Required validates that a value is not empty.
// For strings: checks non-empty
// For slices/maps: checks length > 0
// For pointers: checks not nil
// For bool: always passes (use explicit checks for bool validation)
// For numeric types: always passes (use Min/Max for numeric validation)
func Required() ValidationRule {
	return func(value any) error {
		if value == nil {
			return NewValidationError("is required")
		}

		// Use reflection to handle different types
		v := reflect.ValueOf(value)

		switch v.Kind() {
		case reflect.String:
			if v.String() == "" {
				return NewValidationError("is required")
			}
		case reflect.Slice, reflect.Map, reflect.Array:
			if v.Len() == 0 {
				return NewValidationError("is required")
			}
		case reflect.Ptr:
			if v.IsNil() {
				return NewValidationError("is required")
			}
			// For bool, int, float, etc., we don't check for "empty"
			// since they always have a value (zero value is valid)
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

// Min validates that a numeric value is greater than or equal to min.
// Supports int, int64, float64 types.
func Min[T int | int64 | float64](min T) ValidationRule {
	return func(value any) error {
		switch v := value.(type) {
		case int:
			if T(v) < min {
				return NewValidationError(fmt.Sprintf("must be at least %v", min))
			}
		case int64:
			if T(v) < min {
				return NewValidationError(fmt.Sprintf("must be at least %v", min))
			}
		case float64:
			if T(v) < min {
				return NewValidationError(fmt.Sprintf("must be at least %v", min))
			}
		default:
			return NewValidationError("Min validator only supports int, int64, and float64 types")
		}
		return nil
	}
}

// Max validates that a numeric value is less than or equal to max.
// Supports int, int64, float64 types.
func Max[T int | int64 | float64](max T) ValidationRule {
	return func(value any) error {
		switch v := value.(type) {
		case int:
			if T(v) > max {
				return NewValidationError(fmt.Sprintf("must be at most %v", max))
			}
		case int64:
			if T(v) > max {
				return NewValidationError(fmt.Sprintf("must be at most %v", max))
			}
		case float64:
			if T(v) > max {
				return NewValidationError(fmt.Sprintf("must be at most %v", max))
			}
		default:
			return NewValidationError("Max validator only supports int, int64, and float64 types")
		}
		return nil
	}
}

// Range validates that a numeric value is within the specified range (inclusive).
// Supports int, int64, float64 types.
func Range[T int | int64 | float64](min T, max T) ValidationRule {
	return func(value any) error {
		switch v := value.(type) {
		case int:
			if T(v) < min || T(v) > max {
				return NewValidationError(fmt.Sprintf("must be between %v and %v", min, max))
			}
		case int64:
			if T(v) < min || T(v) > max {
				return NewValidationError(fmt.Sprintf("must be between %v and %v", min, max))
			}
		case float64:
			if T(v) < min || T(v) > max {
				return NewValidationError(fmt.Sprintf("must be between %v and %v", min, max))
			}
		default:
			return NewValidationError("Range validator only supports int, int64, and float64 types")
		}
		return nil
	}
}

// MinItems validates that a slice, array, or map has at least min elements.
// Use AnyLength for no minimum limit.
func MinItems(min int) ValidationRule {
	return func(value any) error {
		if min == AnyLength {
			return nil
		}

		v := reflect.ValueOf(value)

		switch v.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			if v.Len() < min {
				return NewValidationError(fmt.Sprintf("must have at least %d element(s)", min))
			}
		default:
			return NewValidationError("MinItems validator only supports slices, arrays, and maps")
		}

		return nil
	}
}

// MaxItems validates that a slice, array, or map has at most max elements.
// Use AnyLength for no maximum limit.
func MaxItems(max int) ValidationRule {
	return func(value any) error {
		if max == AnyLength {
			return nil
		}

		v := reflect.ValueOf(value)

		switch v.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			if v.Len() > max {
				return NewValidationError(fmt.Sprintf("must have at most %d element(s)", max))
			}
		default:
			return NewValidationError("MaxItems validator only supports slices, arrays, and maps")
		}

		return nil
	}
}

// Each validates each element in a slice or array against the provided rules.
func Each(rules ...ValidationRule) ValidationRule {
	return func(value any) error {
		v := reflect.ValueOf(value)

		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return NewValidationError("Each validator only supports slices and arrays")
		}

		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			for _, rule := range rules {
				if err := rule(elem); err != nil {
					return NewValidationError(fmt.Sprintf("element at index %d: %s", i, err.Error()))
				}
			}
		}

		return nil
	}
}

// Keys validates each key in a map against the provided rules.
func Keys(rules ...ValidationRule) ValidationRule {
	return func(value any) error {
		v := reflect.ValueOf(value)

		if v.Kind() != reflect.Map {
			return NewValidationError("Keys validator only supports maps")
		}

		for _, key := range v.MapKeys() {
			keyValue := key.Interface()
			for _, rule := range rules {
				if err := rule(keyValue); err != nil {
					return NewValidationError(fmt.Sprintf("key '%v': %s", keyValue, err.Error()))
				}
			}
		}

		return nil
	}
}

// Values validates each value in a map against the provided rules.
func Values(rules ...ValidationRule) ValidationRule {
	return func(value any) error {
		v := reflect.ValueOf(value)

		if v.Kind() != reflect.Map {
			return NewValidationError("Values validator only supports maps")
		}

		for _, key := range v.MapKeys() {
			val := v.MapIndex(key).Interface()
			for _, rule := range rules {
				if err := rule(val); err != nil {
					return NewValidationError(fmt.Sprintf("value for key '%v': %s", key.Interface(), err.Error()))
				}
			}
		}

		return nil
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
