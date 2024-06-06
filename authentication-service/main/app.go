package main

import (
	"assignment1/internal/data"
	"assignment1/internal/model"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pressly/goose"
	"github.com/streadway/amqp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "91926499"
	dbname   = "lms_auth"
)

var (
	db             *gorm.DB
	jwtSecret      = []byte("JWT_SECRET")
	tokenExpiresIn = time.Hour * 24
)

var loggerFile *os.File

func init() {
	// Create or open the log file
	var err error
	loggerFile, err = os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Set log output to file
	log.SetOutput(loggerFile)
}

func main() {
	defer loggerFile.Close()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC", host, user, password, dbname, port)
	db := initDB(dsn)

	// Applying migrations
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Ошибка при получении объекта базы данных: %v", err)
	}
	err = goose.Up(sqlDB, "./migrations")
	if err != nil {
		log.Fatalf("Ошибка при применении миграций: %v", err)
	}

	r := setupRoutes(db)
	log.Println("Сервер запущен на :8080")
	http.ListenAndServe(":8080", r)
}

func setupRoutes(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	// Public routes
	r.HandleFunc("/auth/register", RegisterHandler).Methods("POST")
	r.HandleFunc("/auth/login", LoginHandler).Methods("POST")
	r.HandleFunc("/auth/activate/{activationLink}", ActivateHandler).Methods("GET")
	r.HandleFunc("/auth/resend-activation-link", ResendActivationLinkHandler).Methods("GET")

	// Auth required routes
	auth := r.PathPrefix("/auth/api").Subrouter()
	auth.Use(AuthMiddleware())
	auth.HandleFunc("/auth/users", getAllUserInfoHandler).Methods("GET")
	auth.HandleFunc("/auth/users/{id}", getUserInfoHandler).Methods("GET")

	// Admin role required routes
	auth.Use(AdminAuthMiddleware())
	auth.HandleFunc("/auth/admin/users/{id}", editUserInfoHandler).Methods("PUT")
	auth.HandleFunc("/auth/admin/users/{id}", deleteUserInfoHandler).Methods("DELETE")

	// Token validation route
	r.HandleFunc("/auth/validate-token", ValidateTokenHandler).Methods("GET")

	return r
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

func getUserInfoHandler(writer http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)
	userID := params["id"]

	var user data.UserInfo
	if err := db.First(&user, userID).Error; err != nil {
		http.Error(writer, "User not found", http.StatusNotFound)
		log.Printf("User not found: %v", err)
		return
	}

	json.NewEncoder(writer).Encode(user)
}

func editUserInfoHandler(writer http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)
	userID := params["id"]

	var user data.UserInfo
	if err := db.First(&user, userID).Error; err != nil {
		http.Error(writer, "User not found", http.StatusNotFound)
		log.Printf("User not found: %v", err)
		return
	}

	var updatedUser data.UserInfo
	if err := json.NewDecoder(request.Body).Decode(&updatedUser); err != nil {
		http.Error(writer, "Invalid input", http.StatusBadRequest)
		log.Printf("Invalid input: %v", err)
		return
	}

	user.FName = updatedUser.FName
	user.SName = updatedUser.SName
	user.Email = updatedUser.Email
	user.Activated = updatedUser.Activated
	user.UserRole = updatedUser.UserRole

	if err := db.Save(&user).Error; err != nil {
		http.Error(writer, "Failed to update user", http.StatusInternalServerError)
		log.Printf("Failed to update user: %v", err)
		return
	}

	json.NewEncoder(writer).Encode(user)
}

func getAllUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	var users []data.UserInfo
	if err := db.Find(&users).Error; err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		log.Printf("Failed to fetch users: %v", err)
		return
	}

	var usersResponse []map[string]interface{}
	for _, user := range users {
		userResponse := map[string]interface{}{
			"ID":         user.ID,
			"First_name": user.FName,
			"Surname":    user.SName,
			"Email":      user.Email,
			"Activated":  user.Activated,
			"UserRole":   user.UserRole,
		}
		usersResponse = append(usersResponse, userResponse)
	}

	jsonResponse, err := json.Marshal(usersResponse)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var user data.UserInfo
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		log.Printf("Invalid input: %v", err)
		return
	}

	// Check if email already exists
	var existingEmailUser data.UserInfo
	err = db.Where("email = ?", user.Email).First(&existingEmailUser).Error
	if err == nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		log.Printf("Email already exists: %v", err)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(user.PasswordHash, bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		log.Printf("Failed to hash password: %v", err)
		return
	}

	user.ActivationLink = uuid.New().String()
	user.Activated = false
	user.UserRole = "USER"
	user.PasswordHash = hashedPassword

	if err := db.Create(&user).Error; err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		log.Printf("Failed to create user: %v", err)
		return
	}

	// Отправка сообщения на почту
	if err := SendMessageToQueue(user.Email, "Для активации вашей учетной записи перейдите по ссылке: "+user.ActivationLink); err != nil {
		log.Printf("Failed to send activation email to queue: %v", err)
	}

	w.WriteHeader(http.StatusCreated)
}

func ResendActivationLinkHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		log.Println("Missing Authorization header")
		return
	}
	tokenString := authHeader[len("Bearer "):]

	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Printf("Invalid token: %v", err)
		return
	}

	userId := claims.UserId
	if userId == 0 {
		http.Error(w, "UserId is required", http.StatusBadRequest)
		log.Println("UserId is required")
		return
	}

	var user data.UserInfo
	if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		log.Printf("User not found: %v", err)
		return
	}

	newActivationLink := uuid.New().String()

	user.ActivationLink = newActivationLink
	if err := db.Save(&user).Error; err != nil {
		http.Error(w, "Failed to update ActivationLink", http.StatusInternalServerError)
		log.Printf("Failed to update ActivationLink: %v", err)
		return
	}

	jsonResponse, err := json.Marshal(map[string]string{"message": "Activation link resent successfully"})
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	// Отправка сообщения на почту
	if err := SendMessageToQueue(user.Email, "Для активации вашей учетной записи перейдите по ссылке: "+user.ActivationLink); err != nil {
		log.Printf("Failed to send activation email to queue: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var loginRequest model.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginRequest)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		log.Printf("Invalid input: %v", err)
		return
	}

	var user data.UserInfo
	if err := db.Where("email = ?", loginRequest.Email).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		log.Printf("User not found: %v", err)
		return
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(loginRequest.Password)); err != nil {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		log.Printf("Incorrect password: %v", err)
		return
	}

	token, err := GenerateToken(user.ID, user.FName, user.Email, user.Activated, user.UserRole)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		log.Printf("Failed to generate token: %v", err)
		return
	}

	jsonResponse, err := json.Marshal(model.TokenResponse{Token: token})
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)

	// Отправка сообщения на почту о успешном входе
	if err := SendMessageToQueue(user.Email, "Успешный вход в систему"); err != nil {
		log.Printf("Failed to send login email to queue: %v", err)
	}
}

func ActivateHandler(w http.ResponseWriter, r *http.Request) {
	activationLink := mux.Vars(r)["activationLink"]

	var user data.UserInfo
	if err := db.Where("activation_link = ?", activationLink).First(&user).Error; err != nil {
		http.Error(w, "Activation link not found", http.StatusNotFound)
		log.Printf("Activation link not found: %v", err)
		return
	}

	user.Activated = true
	if err := db.Save(&user).Error; err != nil {
		http.Error(w, "Failed to activate user", http.StatusInternalServerError)
		log.Printf("Failed to activate user: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User activated successfully"})

	// Отправка сообщения на почту об успешной активации
	if err := SendMessageToQueue(user.Email, "Ваша учетная запись успешно активирована"); err != nil {
		log.Printf("Failed to send activation email to queue: %v", err)
	}

}

func GenerateToken(userId uint, fname string, email string, isActivated bool, role string) (string, error) {
	expirationTime := time.Now().Add(tokenExpiresIn)
	claims := &model.Claims{
		UserId:      userId,
		Username:    fname,
		IsActivated: isActivated,
		Email:       email,
		ROLE:        role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Error signing token: %v", err)
	}
	return str, err
}

func deleteUserInfoHandler(writer http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)
	userID := params["id"]

	var user data.UserInfo
	if err := db.First(&user, userID).Error; err != nil {
		http.Error(writer, "User not found", http.StatusNotFound)
		log.Printf("User not found: %v", err)
		return
	}

	if err := db.Delete(&user).Error; err != nil {
		http.Error(writer, "Failed to delete user", http.StatusInternalServerError)
		log.Printf("Failed to delete user: %v", err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func AuthMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				log.Println("Missing Authorization header")
				return
			}
			tokenString := authHeader[len("Bearer "):]

			claims := &model.Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				log.Printf("Invalid token: %v", err)
				return
			}

			if !claims.IsActivated {
				http.Error(w, "User not activated", http.StatusForbidden)
				log.Println("User not activated")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AdminAuthMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				log.Println("Missing Authorization header")
				return
			}
			tokenString := authHeader[len("Bearer "):]

			claims := &model.Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				log.Printf("Invalid token: %v", err)
				return
			}

			if claims.ROLE != "ADMIN" {
				http.Error(w, "Unauthorized", http.StatusForbidden)
				log.Println("Unauthorized access attempt")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ValidateTokenHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		log.Println("Missing Authorization header")
		return
	}
	tokenString := authHeader[len("Bearer "):]

	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Printf("Invalid token: %v", err)
		return
	}

	jsonResponse, err := json.Marshal(map[string]interface{}{
		"message": "Token is valid",
		"user": map[string]interface{}{
			"UserId":      claims.UserId,
			"Username":    claims.Username,
			"Email":       claims.Email,
			"IsActivated": claims.IsActivated,
			"ROLE":        claims.ROLE,
		},
	})
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func SendMessageToQueue(messageTo, content string) error {
	// Соединение с RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	// Открытие канала
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	// Объявление очереди
	q, err := ch.QueueDeclare(
		"notification_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	// Создание сообщения в формате JSON
	message := map[string]string{
		"messageTo": messageTo,
		"content":   content,
	}
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Отправка сообщения в очередь
	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to queue: %w", err)
	}

	return nil
}
