package handlers

import (
	"github.com/gin-gonic/gin"
	"lms-crud-api/internal/data"
	"lms-crud-api/internal/helpers"
	"net/http"
)

type CoursesHandler struct {
	Models data.Models
}

func (h *CoursesHandler) CreateCourseHandler(c *gin.Context) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := c.BindJSON(&input); err != nil {
		helpers.BadRequestResponse(c, err)
		return
	}

	course := &data.Course{
		Title:       input.Title,
		Description: input.Description,
	}

	if err := h.Models.Courses.Insert(course); err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusCreated, gin.H{"course": course})
}

func (h *CoursesHandler) ShowAllCoursesHandler(c *gin.Context) {
	courses, err := h.Models.Courses.GetAllWithModulesAndLessons() // Update GetAllWithModulesAndLessons to preload modules and lessons
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"courses": courses})
}

func (h *CoursesHandler) ShowCourseHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	course, err := h.Models.Courses.GetWithModulesAndLessons(id) // Update GetWithModulesAndLessons to preload modules and lessons
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"course": course})
}

func (h *CoursesHandler) UpdateCourseHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	err = c.BindJSON(&input)
	if err != nil {
		helpers.BadRequestResponse(c, err)
		return
	}

	course, err := h.Models.Courses.Get(id)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	course.Title = input.Title
	course.Description = input.Description

	err = h.Models.Courses.Update(course)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"course": course})
}

func (h *CoursesHandler) DeleteCourseHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	err = h.Models.Courses.Delete(id)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
