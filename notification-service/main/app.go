package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/jordan-wright/email"
	"github.com/pressly/goose"
	"github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"time"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "Infinitive"
	dbname   = "notification-service"
)

var (
	db             *gorm.DB
	jwtSecret      = []byte("JWT_SECRET")
	tokenExpiresIn = time.Hour * 24
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MessageTo string    `json:"messageTo"`
	Content   string    `json:"content"`
	SentAt    time.Time `json:"sentAt"`
}

func main() {
	// Set up logging to a file
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	log.Println("Application started")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC", host, user, password, dbname, port)
	db := initDB(dsn)

	// Loading .env file
	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	log.Println("Loaded .env file successfully")

	// Applying migrations
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Ошибка при получении объекта базы данных: %v", err)
	}
	err = goose.Up(sqlDB, "./migrations")
	if err != nil {
		log.Fatalf("Ошибка при применении миграций: %v", err)
	}
	log.Println("Database migrations applied successfully")

	// Auto migrate Notification model
	db.AutoMigrate(&Notification{})
	log.Println("Database migrated")

	// RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	log.Println("Connected to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	log.Println("RabbitMQ channel opened")

	q, err := ch.QueueDeclare(
		"notification_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")
	log.Printf("Declared RabbitMQ queue: %s", q.Name)

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")
	log.Println("Registered RabbitMQ consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a notification: %s", d.Body)

			var notification Notification
			err := json.Unmarshal(d.Body, &notification)
			if err != nil {
				log.Printf("Error decoding JSON: %v", err)
				continue
			}
			log.Printf("Decoded notification - MessageTo: %s, Content: %s", notification.MessageTo, notification.Content)

			err = SendEmail(notification.MessageTo, notification.Content)
			if err != nil {
				log.Printf("Error sending email: %v", err)
			} else {
				log.Printf("Email sent successfully to: %s", notification.MessageTo)
				notification.SentAt = time.Now()
				db.Create(&notification)
			}
		}
	}()

	log.Printf(" [*] Waiting for notifications. To exit press CTRL+C")

	// HTTP
	http.HandleFunc("/notifications", handleNotifications)
	http.HandleFunc("/notifications/", handleNotificationByID)

	log.Println("Server started on :4090")
	log.Fatal(http.ListenAndServe(":4090", nil))
	<-forever
}

func initDB(dsn string) *gorm.DB {
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	log.Println("Database connection established")
	return db
}

func SendEmail(to, content string) error {
	from := os.Getenv("SMTP_MAIL")
	pass := os.Getenv("SMTP_KEY")

	e := email.NewEmail()
	e.From = from
	e.To = []string{to}
	e.Subject = "Notification"
	e.Text = []byte(content)

	log.Printf("Sending email - From: %s, To: %s, Subject: %s", from, to, e.Subject)
	return e.Send("smtp.gmail.com:587", smtp.PlainAuth("", from, pass, "smtp.gmail.com"))
}

// handleNotifications handles the requests for the /notifications endpoint.
func handleNotifications(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getNotifications(w, r)
	case http.MethodPost:
		createNotification(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNotificationByID handles the requests for the /notifications/{id} endpoint.
func handleNotificationByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/notifications/"):]
	switch r.Method {
	case http.MethodGet:
		getNotificationByID(w, r, id)
	case http.MethodPut:
		updateNotification(w, r, id)
	case http.MethodDelete:
		deleteNotification(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getNotifications retrieves all notifications.
func getNotifications(w http.ResponseWriter, r *http.Request) {
	var notifications []Notification
	if err := db.Find(&notifications).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func createNotification(w http.ResponseWriter, r *http.Request) {
	var notification Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	notification.SentAt = time.Now()
	if err := db.Create(&notification).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notification)
}

// getNotificationByID retrieves a notification by ID.
func getNotificationByID(w http.ResponseWriter, r *http.Request, id string) {
	var notification Notification
	if err := db.First(&notification, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// updateNotification updates a notification by ID.
func updateNotification(w http.ResponseWriter, r *http.Request, id string) {
	var notification Notification
	if err := db.First(&notification, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := db.Save(&notification).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// deleteNotification deletes a notification by ID.
func deleteNotification(w http.ResponseWriter, r *http.Request, id string) {
	var notification Notification
	if err := db.First(&notification, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := db.Delete(&notification, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
