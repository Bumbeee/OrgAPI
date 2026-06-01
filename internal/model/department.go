package model

import "time"

type Department struct {
	ID        int          `json:"id" gorm:"primaryKey"`
	Name      string       `json:"name" validate:"required,min=1,max=200" gorm:"type:text;not null"`
	ParentID  *int         `json:"parent_id" gorm:"column:parent_id;index"`
	Employees []Employee   `json:"employees" gorm:"foreignKey:DepartmentID"`
	Children  []Department `json:"children" gorm:"foreignKey:ParentID"`
	CreatedAt time.Time    `json:"created_at" gorm:"column:created_at;type:timestamptz;not null;autoCreateTime"`
}
