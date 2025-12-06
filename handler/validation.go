package handler

import (
	"fmt"
	"regexp"
)

type ValidationError struct {
	Message string
}

func NewValidationError(message string) ValidationError {
	return ValidationError{Message: message}
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
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
