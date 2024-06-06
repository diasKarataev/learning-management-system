package data

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/rs/zerolog"
	"github.com/streadway/amqp"
	"lms-crud-api/middleware"
)

type Notification struct {
	MessageTo string `json:"messageTo"`
	Content   string `json:"content"`
}

type Models struct {
	Courses  CourseModel
	Modules  ModuleModel
	Lessons  LessonModel
	UserInfo UserModel
}

func NewModels(db *gorm.DB) Models {
	return Models{
		Courses:  CourseModel{DB: db},
		Modules:  ModuleModel{DB: db},
		Lessons:  LessonModel{DB: db},
		UserInfo: UserModel{DB: db},
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

func GetUserEmail(c *gin.Context, logger zerolog.Logger) string {
	claims, _ := c.Get("claims")
	userClaims := claims.(*middleware.Claims)

	logger.Info().Msgf("User email: %d", userClaims.Email)

	return userClaims.Email
}

func ConnectToRabbitMQ() (*amqp.Channel, error) {
	conn, err := amqp.Dial("amqp://localhost:5672/") // Adjust connection details if needed
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return ch, nil
}

func DeclareQueue(ch *amqp.Channel, queueName string) error {
	_, err := ch.QueueDeclare(
		"notification_queue", // Name of the queue
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	return nil
}

func PublishMessage(ch *amqp.Channel, queueName string, message string) error {
	err := ch.Publish(
		"",                   // Empty exchange name for routing to the declared queue
		"notification_queue", // Target queue name
		false,                // Mandatory flag (optional, can be set to true for guaranteed delivery)
		false,                // Immediate flag (optional, can be set to true to skip the publisher confirmation)
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(message),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (m CourseModel) SendMessageToQueue(l zerolog.Logger, ch *amqp.Channel, c *gin.Context, content string) {
	// Get the current user's email from the context
	email := GetUserEmail(c, l)

	// Create the notification message
	notification := Notification{
		MessageTo: email,
		Content:   content,
	}

	// Convert the notification message to JSON
	messageJSON, err := json.Marshal(notification)
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to marshal message to JSON")
		return
	}

	// Declare the queue
	err = DeclareQueue(ch, "notification_queue")
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to declare a queue")
		return
	}
	defer ch.Close()

	// Publish the message
	err = PublishMessage(ch, "notification_queue", string(messageJSON))
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to publish a message")
	}
}
