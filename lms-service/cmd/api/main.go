package main

import (
	"fmt"
	"lms-crud-api/cmd/api/handlers"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/rs/zerolog"
	"lms-crud-api/internal/data"
)

type config struct {
	port int
	env  string
	db   struct {
		dsn string
	}
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
	cfg.db.dsn = "postgres://postgres:91926499@localhost/lmscrud?sslmode=disable"

	db, err := openDB(cfg)
	if err != nil {
		log.Fatalln("Failed to connect to database")
	}
	defer db.Close()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	router := gin.Default()

	coursesHandler := &handlers.CoursesHandler{Models: app.models}
	router.POST("/v1/courses", coursesHandler.CreateCourseHandler)
	router.GET("/v1/courses", coursesHandler.ShowAllCoursesHandler)
	router.GET("/v1/courses/:id", coursesHandler.ShowCourseHandler)
	router.PUT("/v1/courses/:id", coursesHandler.UpdateCourseHandler)
	router.DELETE("/v1/courses/:id", coursesHandler.DeleteCourseHandler)

	modulesHandler := &handlers.ModulesHandler{Models: app.models}
	router.POST("/v1/modules", modulesHandler.CreateModuleHandler)
	router.GET("/v1/modules/course/:id", modulesHandler.ShowModulesForCourseHandler)
	router.GET("/v1/modules", modulesHandler.ShowAllModulesHandler)
	router.GET("/v1/modules/:id", modulesHandler.ShowModuleHandler)
	router.PUT("/v1/modules/:id", modulesHandler.UpdateModuleHandler)
	router.DELETE("/v1/modules/:id", modulesHandler.DeleteModuleHandler)

	lessonsHandler := &handlers.LessonsHandler{Models: app.models}
	router.POST("/v1/lessons", lessonsHandler.CreateLessonHandler)
	router.GET("/v1/lessons/module/:id", lessonsHandler.ShowAllLessonsForModuleHandler)
	router.GET("/v1/lessons/:id", lessonsHandler.ShowLessonHandler)
	router.PUT("/v1/lessons/:id", lessonsHandler.UpdateLessonHandler)
	router.DELETE("/v1/lessons/:id", lessonsHandler.DeleteLessonHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

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
