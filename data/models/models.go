package models

import "time"

// Model is same as gorm model with json extension defined to support gin rest
type Model struct {
	ID        uint       `gorm:"primary_key;column:id" json:"id"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
}

// TimeP is there since time can't be null in go without a pointer
func TimeP(v time.Time) *time.Time {
	return &v
}

func UintP(v uint) *uint {
	return &v
}
