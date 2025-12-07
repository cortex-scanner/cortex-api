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

	testID := "5a7bdb69-d7d6-482f-a653-2ab01480999f"
	finding := &repository.AssetFinding{
		ID:      testID,
		AssetID: "7761259c-e6dd-4930-946b-ee9975fde3e4",
		Type:    repository.FindingTypePort,
	}

	mockService.On("GetFinding", mock.Anything, testID).Return(finding, nil)

	runner := test.NewTestRunner(h.HandleGet)
	runner.WithPath("id", testID).Run(t).ExpectNoError().ExpectStatusCode(http.StatusOK)
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
