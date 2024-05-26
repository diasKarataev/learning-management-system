package data

import (
	"github.com/jinzhu/gorm"
)

type Module struct {
	gorm.Model
	Title    string
	CourseID uint
	Lessons  []Lesson
}

type ModuleModel struct {
	DB *gorm.DB
}

func (m ModuleModel) Insert(module *Module) error {
	return m.DB.Create(module).Error
}

func (m ModuleModel) Get(id uint) (*Module, error) {
	var module Module
	err := m.DB.First(&module, id).Error
	if err != nil {
		return nil, err
	}
	return &module, nil
}

func (m ModuleModel) Update(module *Module) error {
	return m.DB.Save(module).Error
}

func (m ModuleModel) Delete(id uint) error {
	return m.DB.Delete(&Module{}, id).Error
}
