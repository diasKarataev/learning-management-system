package data

import (
	"github.com/jinzhu/gorm"
)

type Lesson struct {
	gorm.Model
	Title    string
	Link     string
	Conspect string
	ModuleID uint
}

type LessonModel struct {
	DB *gorm.DB
}

func (m LessonModel) Insert(lesson *Lesson) error {
	return m.DB.Create(lesson).Error
}

func (m LessonModel) Get(id uint) (*Lesson, error) {
	var lesson Lesson
	err := m.DB.First(&lesson, id).Error
	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

func (m LessonModel) Update(lesson *Lesson) error {
	return m.DB.Save(lesson).Error
}

func (m LessonModel) Delete(id uint) error {
	return m.DB.Delete(&Lesson{}, id).Error
}
