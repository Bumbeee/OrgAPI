package repository

import (
	"OrgAPI/internal/model"
	"context"

	"gorm.io/gorm"
)

type Employee interface {
	Create(ctx context.Context, employee *model.Employee) error
}

type employeeRepo struct {
	db *gorm.DB
}

func NewEmployeeRepo(db *gorm.DB) Employee {
	return &employeeRepo{db: db}
}

func (r *employeeRepo) Create(ctx context.Context, employee *model.Employee) error {
	if err := r.db.WithContext(ctx).Create(employee).Error; err != nil {
		return mapPostgresError(err)
	}
	return nil
}
