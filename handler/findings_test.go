package handler_test

import (
	"context"
	"cortex/handler"
	"cortex/repository"
	"cortex/service"
	"cortex/test"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockFindingService struct {
	mock.Mock
}

func (m *MockFindingService) CreateFinding(ctx context.Context, opts service.CreateFindingOptions) (*repository.AssetFinding, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.AssetFinding), args.Error(1)
}

func (m *MockFindingService) GetFinding(ctx context.Context, id string) (*repository.AssetFinding, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.AssetFinding), args.Error(1)
}

func TestGetFinding_Success(t *testing.T) {
	mockService := new(MockFindingService)
	h := handler.NewFindingHandler(mockService)

	finding := &repository.AssetFinding{
		ID:      "test-id",
		AssetID: "asset-id",
		Type:    repository.FindingTypePort,
	}

	mockService.On("GetFinding", mock.Anything, "test-id").Return(finding, nil)

	runner := test.NewTestRunner(h.HandleGet)
	runner.WithPath("id", "test-id").Run(t).ExpectNoError().ExpectStatusCode(http.StatusOK)
}

func TestGetFinding_NotFound(t *testing.T) {
	mockService := new(MockFindingService)
	h := handler.NewFindingHandler(mockService)

	mockService.On("GetFinding", mock.Anything, "missing-id").Return(nil, errors.New("not found"))

	runner := test.NewTestRunner(h.HandleGet)
	res := runner.WithPath("id", "missing-id").Run(t)
	if res.Error == nil {
		t.Error("expected error")
	}
}
