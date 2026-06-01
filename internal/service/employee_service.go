package service

import (
	"OrgAPI/internal/model"
	"OrgAPI/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type EmployeeService struct {
	repo repository.Employee
}

type CreateEmployeeDTO struct {
	FullName     string     `json:"full_name"`
	DepartmentID int        `json:"department_id"`
	Position     string     `json:"position"`
	HiredAt      *time.Time `json:"hired_at"`
}

func (dto *CreateEmployeeDTO) UnmarshalJSON(data []byte) error {
	var raw struct {
		FullName string  `json:"full_name"`
		Position string  `json:"position"`
		HiredAt  *string `json:"hired_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	dto.FullName = raw.FullName
	dto.Position = raw.Position
	if raw.HiredAt != nil && *raw.HiredAt != "" {
		hiredAt, err := time.Parse(time.DateOnly, *raw.HiredAt)
		if err != nil {
			return err
		}
		dto.HiredAt = &hiredAt
	}
	return nil
}

func NewEmployeeService(repo repository.Employee) *EmployeeService {
	return &EmployeeService{repo: repo}
}

func (s *EmployeeService) CreateEmployee(ctx context.Context, dto CreateEmployeeDTO) (*model.Employee, error) {
	dto.FullName = strings.TrimSpace(dto.FullName)
	dto.Position = strings.TrimSpace(dto.Position)
	if err := validateCreateEmployeeDTO(dto); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	employee := &model.Employee{
		FullName:     dto.FullName,
		DepartmentID: dto.DepartmentID,
		Position:     dto.Position,
		HiredAt:      dto.HiredAt,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, employee); err != nil {
		return nil, fmt.Errorf("EmployeeService Create: %w", err)
	}
	return employee, nil
}

func validateCreateEmployeeDTO(dto CreateEmployeeDTO) error {
	if len(dto.FullName) < 1 || len(dto.FullName) > 200 || len(dto.Position) < 1 || len(dto.Position) > 200 {
		return ErrEmployeeValidate
	}
	if dto.DepartmentID <= 0 {
		return ErrInvalidID
	}
	return nil
}
