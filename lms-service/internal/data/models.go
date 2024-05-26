package data

import (
	"github.com/jinzhu/gorm"
)

type Models struct {
	Courses CourseModel
	Modules ModuleModel
	Lessons LessonModel
}

func NewModels(db *gorm.DB) Models {
	return Models{
		Courses: CourseModel{DB: db},
		Modules: ModuleModel{DB: db},
		Lessons: LessonModel{DB: db},
	}
}

func (m ModuleModel) GetWithLessons(id uint) (*Module, error) {
	var module Module
	if err := m.DB.Preload("Lessons").First(&module, id).Error; err != nil {
		return nil, err
	}
	return &module, nil
}

func (m CourseModel) GetWithModulesAndLessons(id uint) (*Course, error) {
	var course Course
	if err := m.DB.Preload("Modules", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Lessons")
	}).First(&course, id).Error; err != nil {
		return nil, err
	}
	return &course, nil
}

func (m CourseModel) GetAllWithModulesAndLessons() ([]Course, error) {
	var courses []Course
	if err := m.DB.Preload("Modules", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Lessons")
	}).Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

func (m ModuleModel) GetAllWithLessonsForCourse(courseID uint) ([]Module, error) {
	var modules []Module
	if err := m.DB.Preload("Lessons").Where("course_id = ?", courseID).Find(&modules).Error; err != nil {
		return nil, err
	}
	return modules, nil
}

func (l LessonModel) GetAllForModule(moduleID uint) ([]Lesson, error) {
	var lessons []Lesson
	if err := l.DB.Where("module_id = ?", moduleID).Find(&lessons).Error; err != nil {
		return nil, err
	}
	return lessons, nil
}

func (m ModuleModel) GetAll() ([]Module, error) {
	var modules []Module
	if err := m.DB.Find(&modules).Error; err != nil {
		return nil, err
	}
	return modules, nil
}
