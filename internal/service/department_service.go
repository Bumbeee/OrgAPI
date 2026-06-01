package service

import (
	"OrgAPI/internal/model"
	"OrgAPI/internal/repository"
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	DeleteModeCascade  = "cascade"
	DeleteModeReassign = "reassign"
)

type DepartmentService struct {
	repo repository.Department
}

func NewDepartmentService(repo repository.Department) *DepartmentService {
	return &DepartmentService{repo: repo}
}

type CreateDepartmentDTO struct {
	Name     string `json:"name"`
	ParentID *int   `json:"parent_id"`
}

type GetDepartmentDTO struct {
	ID   int `json:"id"`
	Opts repository.GetDepartmentOptions
}

type UpdateParentDTO struct {
	ID          int     `json:"id"`
	Name        *string `json:"name"`
	ParentID    *int    `json:"parent_id"`
	ParentIDSet bool
}

type DeleteDepartmentDTO struct {
	ID   int `json:"id"`
	Opts repository.DeleteOptions
}

func (s *DepartmentService) CreateDepartment(ctx context.Context, dto CreateDepartmentDTO) (*model.Department, error) {
	dto.Name = strings.TrimSpace(dto.Name)
	if err := validateCreateDepartmentDTO(dto); err != nil {
		return nil, fmt.Errorf("DepartmentService validateCreateDepartmentDTO: %w", err)
	}

	department := &model.Department{
		Name:      dto.Name,
		ParentID:  dto.ParentID,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, department); err != nil {
		return nil, fmt.Errorf("DepartmentService CreateDepartment: %w", err)
	}
	return department, nil
}

func (s *DepartmentService) GetByID(ctx context.Context, dto GetDepartmentDTO) (*model.Department, error) {
	if err := validateGetDepartmentDTO(dto); err != nil {
		return nil, fmt.Errorf("DepartmentService validateGetDepartmentDTO: %w", err)
	}

	department, err := s.repo.GetByID(ctx, dto.ID, dto.Opts)
	if err != nil {
		return nil, fmt.Errorf("DepartmentService GetByID: %w", err)
	}
	return department, nil
}

func (s *DepartmentService) UpdateParent(ctx context.Context, dto UpdateParentDTO) (*model.Department, error) {
	if dto.Name != nil {
		name := strings.TrimSpace(*dto.Name)
		dto.Name = &name
	}
	if err := validateUpdateParentDTO(dto); err != nil {
		return nil, fmt.Errorf("DepartmentService validateUpdateParentDTO: %w", err)
	}

	dept, err := s.repo.GetByID(ctx, dto.ID, repository.GetDepartmentOptions{Depth: 1, IncludeEmployees: false})
	if err != nil || dept == nil {
		return nil, repository.ErrNotFound
	}

	if dto.Name != nil {
		dept.Name = *dto.Name
	}

	if dto.ParentIDSet {
		if dto.ParentID != nil && *dto.ParentID == dto.ID {
			return nil, ErrSelfParent
		}

		if dto.ParentID != nil {
			parents, err := s.repo.GetParents(ctx, *dto.ParentID)
			if err != nil {
				return nil, err
			}

			for _, parentID := range parents {
				if parentID == dto.ID {
					return nil, ErrCyclicDependency
				}
			}
		}

		dept.ParentID = dto.ParentID
	}

	if err := s.repo.Update(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *DepartmentService) Delete(ctx context.Context, dto DeleteDepartmentDTO) error {
	if err := validateDeleteDepartmentDTO(dto); err != nil {
		return fmt.Errorf("DepartmentService validateDeleteDepartmentDTO: %w", err)
	}

	if err := s.repo.Delete(ctx, dto.ID, dto.Opts); err != nil {
		return fmt.Errorf("DepartmentService Delete: %w", err)
	}
	return nil
}

func validateCreateDepartmentDTO(dto CreateDepartmentDTO) error {
	if len(dto.Name) < 1 || len(dto.Name) > 200 {
		return ErrInvalidParams
	}
	if dto.ParentID != nil && *dto.ParentID <= 0 {
		return ErrInvalidID
	}
	return nil
}

func validateGetDepartmentDTO(dto GetDepartmentDTO) error {
	if dto.ID <= 0 {
		return ErrInvalidID
	}
	if dto.Opts.Depth < 0 || dto.Opts.Depth > 5 {
		return ErrInvalidParams
	}
	return nil
}

func validateUpdateParentDTO(dto UpdateParentDTO) error {
	if dto.ID <= 0 || (dto.ParentID != nil && *dto.ParentID <= 0) {
		return ErrInvalidID
	}
	if dto.ParentID != nil && *dto.ParentID == dto.ID {
		return ErrInvalidParams
	}
	if dto.Name != nil && (len(*dto.Name) < 1 || len(*dto.Name) > 200) {
		return ErrInvalidParams
	}
	return nil
}

func validateDeleteDepartmentDTO(dto DeleteDepartmentDTO) error {
	if dto.ID <= 0 {
		return ErrInvalidID
	}

	mode := dto.Opts.Mode

	switch mode {
	case DeleteModeCascade:
		return nil
	case DeleteModeReassign:
		if dto.Opts.ReassingToID <= 0 {
			return ErrInvalidParams
		}
		return nil
	default:
		return ErrInvalidParams
	}
}
