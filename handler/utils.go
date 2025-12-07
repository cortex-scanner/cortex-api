package handler

import (
	cortexContext "cortex/context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

/********** Responses **********/

type ArrayDataResponse[T any] struct {
	ID         string               `json:"id"`
	APIVersion int                  `json:"apiVersion"`
	Data       APIComponentArray[T] `json:"data"`
}

//nolint:unused // will be used in the future
func newArrayDataResponse[T any](id string, data []T) ArrayDataResponse[T] {
	dataList := data
	if dataList == nil {
		dataList = []T{}
	}

	return ArrayDataResponse[T]{
		ID:         id,
		APIVersion: 1,
		Data: APIComponentArray[T]{
			TotalItems:       len(data),
			Items:            dataList,
			StartIndex:       0,
			CurrentItemCount: len(data),
		},
	}
}

type SingleDataResponse[T any] struct {
	ID         string `json:"id"`
	APIVersion int    `json:"apiVersion"`
	Data       T      `json:"data"`
}

func NewSingleDataResponse[T any](id string, d T) SingleDataResponse[T] {
	return SingleDataResponse[T]{
		ID:         id,
		APIVersion: 1,
		Data:       d,
	}
}

type ErrorResponse struct {
	ID         string             `json:"id"`
	APIVersion int                `json:"apiVersion"`
	Error      ErrorResponseValue `json:"error"`
}

type ErrorResponseValue struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Errors  []ErrorResponseStack `json:"errors"`
}

type ErrorResponseStack struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

func newErrorResponse(id string, code int, message string, errors []error) ErrorResponse {
	resp := ErrorResponse{
		ID:         id,
		APIVersion: 1,
		Error: ErrorResponseValue{
			Code:    code,
			Message: message,
			Errors:  []ErrorResponseStack{},
		},
	}

	for _, e := range errors {
		resp.Error.Errors = append(resp.Error.Errors, ErrorResponseStack{
			Message: e.Error(),
			Reason:  e.Error(),
		})
	}

	return resp
}

//nolint:unused // will be used in the future
type APIComponentArray[T any] struct {
	CurrentItemCount int `json:"currentItemCount"`
	StartIndex       int `json:"startIndex"`
	TotalItems       int `json:"totalItems"`
	Items            []T `json:"items"`
}

/********** API Errors **********/

type APIError struct {
	StatusCode int
	Message    string
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error: %s", e.Message)
}

func NotFound(objectType string, objectID string) APIError {
	return APIError{
		StatusCode: http.StatusNotFound,
		Message:    fmt.Sprintf("%s with id %s not found", objectType, objectID),
	}
}

func OtherError(err error) APIError {
	return APIError{
		StatusCode: http.StatusInternalServerError,
		Message:    err.Error(),
	}
}

/********** API Functions **********/

type APIFunc func(w http.ResponseWriter, r *http.Request) error

func Make(f APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			var apiErr APIError
			if errors.As(err, &apiErr) {
				RespondError(w, r, apiErr.StatusCode, err)
			} else {
				// unknown error type, respond with internal server error
				RespondError(w, r, http.StatusInternalServerError, err)
			}
		}
	}
}

/********** Utility functions **********/

func RespondError(w http.ResponseWriter, r *http.Request, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errorReply := newErrorResponse(cortexContext.RequestID(r.Context()), status, err.Error(), nil)
	e := json.NewEncoder(w).Encode(errorReply)
	if e != nil {
		panic(err)
	}
}

func RespondOne[T any](w http.ResponseWriter, r *http.Request, data T) error {
	return respondOneWithStatus(w, r, http.StatusOK, data)
}

//nolint:unused // will be used in the future
func RespondOneCreated[T any](w http.ResponseWriter, r *http.Request, data T) error {
	return respondOneWithStatus(w, r, http.StatusCreated, data)
}

func RespondMany[T any](w http.ResponseWriter, r *http.Request, data []T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := newArrayDataResponse(cortexContext.RequestID(r.Context()), data)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return err
	}

	return nil
}

func respondOneWithStatus[T any](w http.ResponseWriter, r *http.Request, status int, data T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := NewSingleDataResponse(cortexContext.RequestID(r.Context()), data)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return err
	}

	return nil
}

func ValidateParam(r *http.Request, param string) (string, error) {
	return ValidateString(r.PathValue(param), UUID()).Validate()
}

func WrapError(err error) APIError {
	var apiErr APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	var validationErr ValidationError
	if errors.As(err, &validationErr) {
		return APIError{
			StatusCode: http.StatusBadRequest,
			Message:    validationErr.Error(),
		}
	}

	// TODO: handle other cases like not found, unique violation, etc.
	return OtherError(err)
}
