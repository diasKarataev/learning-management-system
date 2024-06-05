package main

import (
	"fmt"
	"lms-crud-api/internal/helpers"
	"lms-crud-api/middleware"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pressly/goose"
	"github.com/rs/zerolog"
	"lms-crud-api/cmd/api/handlers"
	"lms-crud-api/internal/data"
)

type config struct {
	port int
	env  string
	db   struct {
		dsn string
	}
	secretKey string
}

type application struct {
	config config
	logger zerolog.Logger
	models data.Models
}

func main() {
	var cfg config
	cfg.port = 4000
	cfg.env = "development"
	cfg.db.dsn = "postgres://postgres:Infinitive@localhost/lms-service?sslmode=disable"
	cfg.secretKey = "JWT_SECRET"

	db, err := openDB(cfg)
	if err != nil {
		log.Fatalln("Failed to connect to database")
	}
	defer db.Close()

	// Applying migrations
	sqlDB := db.DB()
	if err != nil {
		log.Fatalf("Error getting DB object: %v", err)
	}
	err = goose.Up(sqlDB, "./migrations")
	if err != nil {
		log.Fatalf("Error applying migrations: %v", err)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	router := gin.Default()

	authMiddleware := middleware.AuthMiddleware(cfg.secretKey)

	coursesHandler := &handlers.CoursesHandler{Models: app.models}
	router.POST("/lms/courses", authMiddleware, coursesHandler.CreateCourseHandler)
	router.GET("/api/lms/courses", authMiddleware, coursesHandler.ShowAllCoursesHandler)
	router.GET("/lms/courses/:id", authMiddleware, coursesHandler.ShowCourseHandler)
	router.PUT("/lms/courses/:id", authMiddleware, coursesHandler.UpdateCourseHandler)
	router.DELETE("/lms/courses/:id", authMiddleware, coursesHandler.DeleteCourseHandler)

	modulesHandler := &handlers.ModulesHandler{Models: app.models}
	router.POST("/lms/modules", authMiddleware, modulesHandler.CreateModuleHandler)
	router.GET("/lms/modules/course/:id", authMiddleware, modulesHandler.ShowModulesForCourseHandler)
	router.GET("/lms/modules", authMiddleware, modulesHandler.ShowAllModulesHandler)
	router.GET("/lms/modules/:id", authMiddleware, modulesHandler.ShowModuleHandler)
	router.PUT("/lms/modules/:id", authMiddleware, modulesHandler.UpdateModuleHandler)
	router.DELETE("/lms/modules/:id", authMiddleware, modulesHandler.DeleteModuleHandler)

	lessonsHandler := &handlers.LessonsHandler{Models: app.models}
	router.POST("/lms/lessons", authMiddleware, lessonsHandler.CreateLessonHandler)
	router.GET("/lms/lessons/module/:id", authMiddleware, lessonsHandler.ShowAllLessonsForModuleHandler)
	router.GET("/lms/lessons/:id", authMiddleware, lessonsHandler.ShowLessonHandler)
	router.PUT("/lms/lessons/:id", authMiddleware, lessonsHandler.UpdateLessonHandler)
	router.DELETE("/lms/lessons/:id", authMiddleware, lessonsHandler.DeleteLessonHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	ch, _ := helpers.ConnectToRabbitMQ()
	err = helpers.DeclareQueue(ch, "notification_queue")
	if err != nil {
		logger.Fatal().Err(err).Msg("Could not declare queue")
	}
	defer ch.Close()

	logger.Info().Msgf("Starting server on %s", srv.Addr)
	err = srv.ListenAndServe()
	if err != nil {
		logger.Fatal().Err(err).Msg("Could not start server")
	}
}

func openDB(cfg config) (*gorm.DB, error) {
	db, err := gorm.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}
