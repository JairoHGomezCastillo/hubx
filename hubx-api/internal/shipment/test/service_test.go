package test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/shipment"
	"hubx-api/internal/shipment/mocks"
)

func TestHU02_Create_ForbiddenRole_MakesNoRepositoryCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	codeGen := mocks.NewMockCodeGenerator(ctrl)

	// No repository expectations: service should return forbidden without calling repo

	service := shipment.NewShipmentService(repo, codeGen)

	input := shipment.CreateInput{
		Role:          "ground partner",
		OperatorID:    "some-id",
		Origin:        "Bogotá",
		Destination:   "Miami",
		Flight:        "AV-204",
		WeightKg:      "120",
		CustomerEmail: "andina@hubx.co",
	}

	_, err := service.Create(context.Background(), input)
	if !errors.Is(err, shipment.ErrForbidden) {
		t.Errorf("Create() with ground partner role = %v; want ErrForbidden", err)
	}
}

func TestHU02_Create_CustomerRole_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	codeGen := mocks.NewMockCodeGenerator(ctrl)
	service := shipment.NewShipmentService(repo, codeGen)

	input := shipment.CreateInput{
		Role:          "customer",
		Origin:        "Bogotá",
		Destination:   "Miami",
		Flight:        "AV-204",
		WeightKg:      "120",
		CustomerEmail: "andina@hubx.co",
	}

	_, err := service.Create(context.Background(), input)
	if !errors.Is(err, shipment.ErrForbidden) {
		t.Errorf("Create() with customer role = %v; want ErrForbidden", err)
	}
}

func TestHU02_Create_MissingFields(t *testing.T) {
	tests := []struct {
		name  string
		input shipment.CreateInput
	}{
		{
			name: "missing origin",
			input: shipment.CreateInput{
				Role:          "operator",
				Origin:        "",
				Destination:   "Miami",
				Flight:        "AV-204",
				WeightKg:      "120",
				CustomerEmail: "andina@hubx.co",
			},
		},
		{
			name: "blank origin",
			input: shipment.CreateInput{
				Role:          "operator",
				Origin:        "   ",
				Destination:   "Miami",
				Flight:        "AV-204",
				WeightKg:      "120",
				CustomerEmail: "andina@hubx.co",
			},
		},
		{
			name: "missing destination",
			input: shipment.CreateInput{
				Role:          "operator",
				Origin:        "Bogotá",
				Destination:   "",
				Flight:        "AV-204",
				WeightKg:      "120",
				CustomerEmail: "andina@hubx.co",
			},
		},
		{
			name: "missing flight",
			input: shipment.CreateInput{
				Role:          "operator",
				Origin:        "Bogotá",
				Destination:   "Miami",
				Flight:        "",
				WeightKg:      "120",
				CustomerEmail: "andina@hubx.co",
			},
		},
		{
			name: "missing weight",
			input: shipment.CreateInput{
				Role:          "operator",
				Origin:        "Bogotá",
				Destination:   "Miami",
				Flight:        "AV-204",
				WeightKg:      "",
				CustomerEmail: "andina@hubx.co",
			},
		},
		{
			name: "missing customer email",
			input: shipment.CreateInput{
				Role:          "operator",
				Origin:        "Bogotá",
				Destination:   "Miami",
				Flight:        "AV-204",
				WeightKg:      "120",
				CustomerEmail: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockRepository(ctrl)
			codeGen := mocks.NewMockCodeGenerator(ctrl)
			service := shipment.NewShipmentService(repo, codeGen)

			_, err := service.Create(context.Background(), tt.input)
			if !errors.Is(err, shipment.ErrMissingData) {
				t.Errorf("Create() = %v; want ErrMissingData", err)
			}
		})
	}
}

func TestHU02_Create_SameOriginDestination(t *testing.T) {
	tests := []struct {
		name        string
		origin      string
		destination string
	}{
		{
			name:        "exact match",
			origin:      "Bogotá",
			destination: "Bogotá",
		},
		{
			name:        "case insensitive",
			origin:      "BOGOTÁ",
			destination: "bogotá",
		},
		{
			name:        "with spaces",
			origin:      "  Bogotá  ",
			destination: "Bogotá",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockRepository(ctrl)
			codeGen := mocks.NewMockCodeGenerator(ctrl)
			service := shipment.NewShipmentService(repo, codeGen)

			input := shipment.CreateInput{
				Role:          "operator",
				Origin:        tt.origin,
				Destination:   tt.destination,
				Flight:        "AV-204",
				WeightKg:      "120",
				CustomerEmail: "andina@hubx.co",
			}

			_, err := service.Create(context.Background(), input)
			if !errors.Is(err, shipment.ErrSameOriginDestination) {
				t.Errorf("Create() = %v; want ErrSameOriginDestination", err)
			}
		})
	}
}

func TestHU02_Create_WeightValidation(t *testing.T) {
	tests := []struct {
		name     string
		weightKg string
		wantErr  error
	}{
		{
			name:     "below minimum",
			weightKg: "0",
			wantErr:  shipment.ErrWeightOutOfRange,
		},
		{
			name:     "negative weight",
			weightKg: "-1",
			wantErr:  shipment.ErrWeightOutOfRange,
		},
		{
			name:     "at minimum",
			weightKg: "0.1",
			wantErr:  nil,
		},
		{
			name:     "at maximum",
			weightKg: "5000",
			wantErr:  nil,
		},
		{
			name:     "above maximum",
			weightKg: "5001",
			wantErr:  shipment.ErrWeightOutOfRange,
		},
		{
			name:     "one decimal valid",
			weightKg: "0.5",
			wantErr:  nil,
		},
		{
			name:     "two decimals invalid",
			weightKg: "0.15",
			wantErr:  shipment.ErrWeightOutOfRange,
		},
		{
			name:     "not a number",
			weightKg: "abc",
			wantErr:  shipment.ErrWeightOutOfRange,
		},
		{
			name:     "exponent notation 5e3",
			weightKg: "5e3",
			wantErr:  shipment.ErrWeightOutOfRange,
		},
		{
			name:     "exponent notation 1e-1",
			weightKg: "1e-1",
			wantErr:  shipment.ErrWeightOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockRepository(ctrl)
			codeGen := mocks.NewMockCodeGenerator(ctrl)

			if tt.wantErr == nil {
				// Valid weight: expect repository calls
				repo.EXPECT().
					FindUserByEmail(gomock.Any(), "andina@hubx.co").
					Return(&shipment.Customer{
						ID:    "cust-id",
						Email: "andina@hubx.co",
						Name:  "Distribuciones Andina",
						Role:  "customer",
					}, nil)

				codeGen.EXPECT().Generate().Return("HBX-12345678")

				repo.EXPECT().
					Insert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			}

			service := shipment.NewShipmentService(repo, codeGen)

			input := shipment.CreateInput{
				Role:          "operator",
				OperatorID:    "op-id",
				Origin:        "Bogotá",
				Destination:   "Miami",
				Flight:        "AV-204",
				WeightKg:      tt.weightKg,
				CustomerEmail: "andina@hubx.co",
			}

			_, err := service.Create(context.Background(), input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Create() = %v; want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Create() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestHU02_Create_CustomerNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	codeGen := mocks.NewMockCodeGenerator(ctrl)

	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "nobody@hubx.co").
		Return(nil, shipment.ErrCustomerNotFound)

	service := shipment.NewShipmentService(repo, codeGen)

	input := shipment.CreateInput{
		Role:          "operator",
		Origin:        "Bogotá",
		Destination:   "Miami",
		Flight:        "AV-204",
		WeightKg:      "120",
		CustomerEmail: "nobody@hubx.co",
	}

	_, err := service.Create(context.Background(), input)
	if !errors.Is(err, shipment.ErrCustomerNotFound) {
		t.Errorf("Create() = %v; want ErrCustomerNotFound", err)
	}
}

func TestHU02_Create_NotACustomerRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	codeGen := mocks.NewMockCodeGenerator(ctrl)

	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "rapido@hubx.co").
		Return(&shipment.Customer{
			ID:    "partner-id",
			Email: "rapido@hubx.co",
			Name:  "Rápido Express",
			Role:  "ground partner",
		}, nil)

	service := shipment.NewShipmentService(repo, codeGen)

	input := shipment.CreateInput{
		Role:          "operator",
		Origin:        "Bogotá",
		Destination:   "Miami",
		Flight:        "AV-204",
		WeightKg:      "120",
		CustomerEmail: "rapido@hubx.co",
	}

	_, err := service.Create(context.Background(), input)
	if !errors.Is(err, shipment.ErrNotACustomer) {
		t.Errorf("Create() = %v; want ErrNotACustomer", err)
	}
}

func TestHU02_Create_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	codeGen := mocks.NewMockCodeGenerator(ctrl)

	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "andina@hubx.co").
		Return(&shipment.Customer{
			ID:    "cust-id",
			Email: "andina@hubx.co",
			Name:  "Distribuciones Andina",
			Role:  "customer",
		}, nil)

	codeGen.EXPECT().Generate().Return("HBX-7F3A9C21")

	repo.EXPECT().
		Insert(gomock.Any(), gomock.Any(), "op-id", "cust-id").
		Return(nil)

	service := shipment.NewShipmentService(repo, codeGen)

	input := shipment.CreateInput{
		Role:          "operator",
		OperatorID:    "op-id",
		Origin:        "Bogotá",
		Destination:   "Miami",
		Flight:        "AV-204",
		WeightKg:      "120",
		CustomerEmail: "andina@hubx.co",
	}

	result, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v; want nil", err)
	}

	if result.Code != "HBX-7F3A9C21" {
		t.Errorf("Create() code = %q; want %q", result.Code, "HBX-7F3A9C21")
	}
	if result.Stage != "Created" {
		t.Errorf("Create() stage = %q; want %q", result.Stage, "Created")
	}
	if result.CustomerName != "Distribuciones Andina" {
		t.Errorf("Create() customer name = %q; want %q", result.CustomerName, "Distribuciones Andina")
	}
}

func TestHU02_Create_DuplicateCodeRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	codeGen := mocks.NewMockCodeGenerator(ctrl)

	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "andina@hubx.co").
		Return(&shipment.Customer{
			ID:    "cust-id",
			Email: "andina@hubx.co",
			Name:  "Distribuciones Andina",
			Role:  "customer",
		}, nil)

	// First attempt: duplicate code (use sentinel)
	codeGen.EXPECT().Generate().Return("HBX-00000001")
	repo.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(shipment.ErrDuplicateCode)

	// Second attempt: success
	codeGen.EXPECT().Generate().Return("HBX-00000002")
	repo.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	service := shipment.NewShipmentService(repo, codeGen)

	input := shipment.CreateInput{
		Role:          "operator",
		OperatorID:    "op-id",
		Origin:        "Bogotá",
		Destination:   "Miami",
		Flight:        "AV-204",
		WeightKg:      "120",
		CustomerEmail: "andina@hubx.co",
	}

	result, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v; want nil", err)
	}

	if result.Code != "HBX-00000002" {
		t.Errorf("Create() code = %q; want %q (second attempt)", result.Code, "HBX-00000002")
	}
}

func TestHU02_Create_ThreeCollisions_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	codeGen := mocks.NewMockCodeGenerator(ctrl)

	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "andina@hubx.co").
		Return(&shipment.Customer{
			ID:    "cust-id",
			Email: "andina@hubx.co",
			Name:  "Distribuciones Andina",
			Role:  "customer",
		}, nil)

	// All three attempts fail (use sentinel)
	codeGen.EXPECT().Generate().Return("HBX-00000001")
	repo.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(shipment.ErrDuplicateCode)

	codeGen.EXPECT().Generate().Return("HBX-00000002")
	repo.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(shipment.ErrDuplicateCode)

	codeGen.EXPECT().Generate().Return("HBX-00000003")
	repo.EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(shipment.ErrDuplicateCode)

	service := shipment.NewShipmentService(repo, codeGen)

	input := shipment.CreateInput{
		Role:          "operator",
		OperatorID:    "op-id",
		Origin:        "Bogotá",
		Destination:   "Miami",
		Flight:        "AV-204",
		WeightKg:      "120",
		CustomerEmail: "andina@hubx.co",
	}

	_, err := service.Create(context.Background(), input)
	// After 3 collisions, should return a wrapped internal error (not the sentinel)
	if err == nil {
		t.Fatal("Create() error = nil; want error after 3 collisions")
	}
	if errors.Is(err, shipment.ErrDuplicateCode) {
		t.Errorf("Create() error should be wrapped, not the sentinel ErrDuplicateCode")
	}
}
