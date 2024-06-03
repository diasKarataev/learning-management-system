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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/http"
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

func main() {
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
	r.HandleFunc("/register", RegisterHandler).Methods("POST")
	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/activate/{activationLink}", ActivateHandler).Methods("GET")
	r.HandleFunc("/resend-activation-link", ResendActivationLinkHandler).Methods("GET")

	// Auth required routes
	auth := r.PathPrefix("/api").Subrouter()
	auth.Use(AuthMiddleware())
	auth.HandleFunc("/users", getAllUserInfoHandler).Methods("GET")
	auth.HandleFunc("/users/{id}", getUserInfoHandler).Methods("GET")

	// Admin role required routes
	auth.Use(AdminAuthMiddleware())
	auth.HandleFunc("/admin/users/{id}", editUserInfoHandler).Methods("PUT")
	auth.HandleFunc("/admin/users/{id}", deleteUserInfoHandler).Methods("DELETE")

	// Token validation route
	r.HandleFunc("/validate-token", ValidateTokenHandler).Methods("GET")

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
		return
	}

	var updatedUser data.UserInfo
	if err := json.NewDecoder(request.Body).Decode(&updatedUser); err != nil {
		http.Error(writer, "Invalid input", http.StatusBadRequest)
		return
	}

	user.FName = updatedUser.FName
	user.SName = updatedUser.SName
	user.Email = updatedUser.Email
	user.Activated = updatedUser.Activated
	user.UserRole = updatedUser.UserRole

	if err := db.Save(&user).Error; err != nil {
		http.Error(writer, "Failed to update user", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(writer).Encode(user)
}

func getAllUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	var users []data.UserInfo
	if err := db.Find(&users).Error; err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
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
		return
	}

	// Check if email already exists
	var existingEmailUser data.UserInfo
	err = db.Where("email = ?", user.Email).First(&existingEmailUser).Error
	if err == nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(user.PasswordHash, bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	user.ActivationLink = uuid.New().String()
	user.Activated = false
	user.UserRole = "USER"
	user.PasswordHash = hashedPassword

	if err := db.Create(&user).Error; err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	if err := SendActivationEmail(user.Email, user.ActivationLink); err != nil {
		http.Error(w, "Failed to send activation email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func SendActivationEmail(to, activationLink string) error {
	// TODO: RAbbit
	return nil
}

func ResendActivationLinkHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}
	tokenString := authHeader[len("Bearer "):]

	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userId := claims.UserId
	if userId == 0 {
		http.Error(w, "UserId is required", http.StatusBadRequest)
		return
	}

	var user data.UserInfo
	if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	newActivationLink := uuid.New().String()

	user.ActivationLink = newActivationLink
	if err := db.Save(&user).Error; err != nil {
		http.Error(w, "Failed to update ActivationLink", http.StatusInternalServerError)
		return
	}

	if err := SendActivationEmail(user.Email, user.ActivationLink); err != nil {
		http.Error(w, "Failed to send activation email", http.StatusInternalServerError)
		return
	}

	jsonResponse, err := json.Marshal(map[string]string{"message": "Activation link resent successfully"})
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
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
		return
	}

	var user data.UserInfo
	if err := db.Where("email = ?", loginRequest.Email).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(loginRequest.Password)); err != nil {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	token, err := GenerateToken(user.ID, user.FName, user.Email, user.Activated, user.UserRole)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	jsonResponse, err := json.Marshal(model.TokenResponse{Token: token})
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func ActivateHandler(w http.ResponseWriter, r *http.Request) {
	activationLink := mux.Vars(r)["activationLink"]

	var user data.UserInfo
	if err := db.Where("activation_link = ?", activationLink).First(&user).Error; err != nil {
		http.Error(w, "Activation link not found", http.StatusNotFound)
		return
	}

	user.Activated = true
	if err := db.Save(&user).Error; err != nil {
		http.Error(w, "Failed to activate user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User activated successfully"})
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
		return
	}

	if err := db.Delete(&user).Error; err != nil {
		http.Error(writer, "Failed to delete user", http.StatusInternalServerError)
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
				return
			}
			tokenString := authHeader[len("Bearer "):]

			claims := &model.Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if !claims.IsActivated {
				http.Error(w, "User not activated", http.StatusForbidden)
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
				return
			}
			tokenString := authHeader[len("Bearer "):]

			claims := &model.Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if claims.ROLE != "ADMIN" {
				http.Error(w, "Unauthorized", http.StatusForbidden)
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
		return
	}
	tokenString := authHeader[len("Bearer "):]

	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
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
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}
