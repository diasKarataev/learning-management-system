package main

import (
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
	dbname   = "d.karataevDB"
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

func main() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC", host, user, password, dbname, port)
	db := initDB(dsn)

	// Loading .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Applying migrations
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Ошибка при получении объекта базы данных: %v", err)
	}
	err = goose.Up(sqlDB, "./migrations")
	if err != nil {
		log.Fatalf("Ошибка при применении миграций: %v", err)
	}

	// RabbitMQ

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"notification_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

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

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a notification: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for notifications. To exit press CTRL+C")
	<-forever

	// HTTP
	log.Println("Сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}
func initDB(dsn string) *gorm.DB {
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	return db
}

func SendEmail(to, activationLink string) error {
	from := "karataev020902@gmail.com"
	pass := os.Getenv("SMTP_KEY")

	e := email.NewEmail()
	e.From = from
	e.To = []string{to}
	e.Subject = "ActivateHandler your account"
	e.HTML = []byte(fmt.Sprintf("Click <a href=\"%s/activate/%s\">here</a> to activate your account", os.Getenv("API_URL"), activationLink))

	return e.Send("smtp.gmail.com:587", smtp.PlainAuth("", from, pass, "smtp.gmail.com"))
}
