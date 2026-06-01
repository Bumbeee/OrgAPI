package model

import "time"

type Employee struct {
	ID           int        `json:"id" gorm:"primaryKey;autoIncrement"`
	DepartmentID int        `json:"department_id" gorm:"column:department_id;not null;index"`
	FullName     string     `json:"full_name" validate:"required,min=1,max=200" gorm:"column:full_name;type:varchar(200);not null"`
	Position     string     `json:"position" validate:"required,min=1,max=200" gorm:"column:position;type:varchar(200);not null"`
	HiredAt      *time.Time `json:"hired_at" gorm:"column:hired_at;type:date;"`
	CreatedAt    time.Time  `json:"created_at" gorm:"column:created_at;type:timestamptz;not null;autoCreateTime"`
}
