package repository

import (
	"OrgAPI/internal/model"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Department interface {
	Create(ctx context.Context, department *model.Department) error
	GetByID(ctx context.Context, id int, options GetDepartmentOptions) (*model.Department, error)
	Update(ctx context.Context, dept *model.Department) error
	Delete(ctx context.Context, id int, opts DeleteOptions) error
	GetParents(ctx context.Context, startID int) ([]int, error)
}

type departmentRepo struct {
	db *gorm.DB
}

type GetDepartmentOptions struct {
	Depth            int
	IncludeEmployees bool
	SortBy           string
}

type DeleteOptions struct {
	Mode         string
	ReassingToID int
}

type UpdateDepartment struct {
	Name        *string
	ParentID    *int
	ParentIDSet bool
}

func NewDepartmentRepo(db *gorm.DB) Department {
	return &departmentRepo{db: db}
}

func (r *departmentRepo) Create(ctx context.Context, department *model.Department) error {
	if err := r.db.WithContext(ctx).Create(department).Error; err != nil {
		return mapPostgresError(err)
	}
	return nil
}

func (r *departmentRepo) GetByID(ctx context.Context, id int, options GetDepartmentOptions) (*model.Department, error) {
	var department model.Department
	query := r.db.WithContext(ctx)

	if options.IncludeEmployees {
		query = query.Preload("Employees", func(db *gorm.DB) *gorm.DB {
			return r.sortEmployees(db, options.SortBy)
		})
	}

	if err := query.First(&department, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("departmentRepo GetDepartmentByID repo: %w", err)
	}

	if options.Depth > 0 {
		if err := r.loadDepartmentChildren(ctx, &department, options, 1); err != nil {
			return nil, err
		}
	}

	return &department, nil
}

func (r *departmentRepo) Update(ctx context.Context, dept *model.Department) error {
	var parentID any
	if dept.ParentID != nil {
		parentID = *dept.ParentID
	} else {
		parentID = gorm.Expr("NULL")
	}

	query := r.db.WithContext(ctx).Model(&model.Department{}).Where("id = ?", dept.ID)
	if err := query.Update("name", dept.Name).Error; err != nil {
		return mapPostgresError(err)
	}
	if err := query.Update("parent_id", parentID).Error; err != nil {
		return mapPostgresError(err)
	}
	return nil
}

func (r *departmentRepo) GetParents(ctx context.Context, startID int) ([]int, error) {
	var IDs []int

	rawQuery := `
		WITH RECURSIVE parents AS (
			SELECT id, parent_id FROM departments WHERE id = ?
			UNION ALL
			SELECT d.id, d.parent_id FROM departments d
			JOIN parents a ON d.id = a.parent_id
		)
		SELECT id FROM parents WHERE id != ?;
	`
	if err := r.db.WithContext(ctx).Raw(rawQuery, startID, startID).Scan(&IDs).Error; err != nil {
		return nil, err
	}
	return IDs, nil
}

func (r *departmentRepo) Delete(ctx context.Context, id int, opts DeleteOptions) error {
	if err := r.db.WithContext(ctx).First(&model.Department{}, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("Delete: %w", err)
	}

	switch opts.Mode {
	case "cascade":
		return r.deleteCascade(ctx, id)
	default:
		return r.deleteReassign(ctx, id, opts)
	}
}

func (r *departmentRepo) deleteCascade(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&model.Department{}, id).Error
}

func (r *departmentRepo) deleteReassign(ctx context.Context, id int, opts DeleteOptions) error {
	if id == opts.ReassingToID {
		return ErrCycle
	}
	if err := r.db.WithContext(ctx).First(&model.Department{}, opts.ReassingToID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("departmentRepo deleteReassign: %w", err)
	}

	var department model.Department
	if err := r.db.WithContext(ctx).First(&department, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("departmentRepo deleteReassign department lookup: %w", err)
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Employee{}).Where("department_id = ?", id).Update("department_id", opts.ReassingToID).Error; err != nil {
			return fmt.Errorf("deleteReassign update: %w", err)
		}

		if err := tx.Model(&model.Department{}).Where("parent_id = ?", id).Update("parent_id", department.ParentID).Error; err != nil {
			return mapPostgresError(err)
		}

		if err := tx.Delete(&model.Department{}, id).Error; err != nil {
			return fmt.Errorf("departmentRepo deleteReassign delete: %w", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("departmentRepo deleteReassign transaction: %w", err)
	}

	return nil
}

func (r *departmentRepo) loadDepartmentChildren(ctx context.Context, parent *model.Department, options GetDepartmentOptions, currentDepth int) error {
	if currentDepth > options.Depth {
		return nil
	}

	query := r.db.WithContext(ctx).Where("parent_id = ?", parent.ID)

	if options.IncludeEmployees {
		query = query.Preload("Employees", func(db *gorm.DB) *gorm.DB {
			return r.sortEmployees(db, options.SortBy)
		})
	}

	if err := query.Find(&parent.Children).Error; err != nil {
		return fmt.Errorf("departmentRepo load children, department: %d: %w", parent.ID, err)
	}

	if currentDepth < options.Depth {
		for i := range parent.Children {
			if err := r.loadDepartmentChildren(ctx, &parent.Children[i], options, currentDepth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *departmentRepo) sortEmployees(db *gorm.DB, sortBy string) *gorm.DB {
	switch sortBy {
	case "full_name":
		return db.Order("full_name ASC")
	default:
		return db.Order("created_at ASC")
	}
}

func mapPostgresError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case "23503":
		return ErrNotFound
	case "23505":
		return ErrAlreadyExists
	default:
		return err
	}
}
