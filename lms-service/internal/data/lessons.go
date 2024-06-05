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

func (m LessonModel) GetModuleName(moduleID int) string {
	var moduleName string
	result := m.DB.Table("modules").Select("title").Where("id = ?", moduleID).First(&moduleName)
	if result.Error != nil {
		return ""
	}
	return moduleName
}

func (m LessonModel) GetCourseNameByModuleId(moduleID int) string {
	var courseName string
	result := m.DB.Table("modules").Select("courses.title").Joins("JOIN courses ON modules.course_id = courses.id").Where("modules.id = ?", moduleID).First(&courseName)
	if result.Error != nil {
		return ""
	}
	return courseName
}
