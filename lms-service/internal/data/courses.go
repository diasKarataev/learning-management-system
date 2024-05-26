package data

import (
	"github.com/jinzhu/gorm"
)

type Course struct {
	gorm.Model
	Title       string
	Description string
	Modules     []Module
}

type CourseModel struct {
	DB *gorm.DB
}

func (m CourseModel) Insert(course *Course) error {
	return m.DB.Create(course).Error
}

func (m CourseModel) Get(id uint) (*Course, error) {
	var course Course
	err := m.DB.First(&course, id).Error
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (m CourseModel) Update(course *Course) error {
	return m.DB.Save(course).Error
}

func (m CourseModel) Delete(id uint) error {
	return m.DB.Delete(&Course{}, id).Error
}
