package handlers

import (
	"github.com/gin-gonic/gin"
	"lms-crud-api/internal/data"
	"lms-crud-api/internal/helpers"
	"net/http"
)

type LessonsHandler struct {
	Models data.Models
}

func (h *LessonsHandler) CreateLessonHandler(c *gin.Context) {
	var input struct {
		Title    string `json:"title"`
		Link     string `json:"link"`
		Conspect string `json:"conspect"`
		ModuleID uint   `json:"module_id"`
	}

	err := c.BindJSON(&input)
	if err != nil {
		helpers.BadRequestResponse(c, err)
		return
	}

	lesson := &data.Lesson{
		Title:    input.Title,
		Link:     input.Link,
		Conspect: input.Conspect,
		ModuleID: input.ModuleID,
	}

	err = h.Models.Lessons.Insert(lesson)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusCreated, gin.H{"lesson": lesson})
}

func (h *LessonsHandler) ShowAllLessonsForModuleHandler(c *gin.Context) {
	moduleID, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	lessons, err := h.Models.Lessons.GetAllForModule(moduleID)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"lessons": lessons})
}

func (h *LessonsHandler) ShowLessonHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	lesson, err := h.Models.Lessons.Get(id)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"lesson": lesson})
}

func (h *LessonsHandler) UpdateLessonHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	var input struct {
		Title    string `json:"title"`
		Link     string `json:"link"`
		Conspect string `json:"conspect"`
	}

	err = c.BindJSON(&input)
	if err != nil {
		helpers.BadRequestResponse(c, err)
		return
	}

	lesson, err := h.Models.Lessons.Get(id)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	lesson.Title = input.Title
	lesson.Link = input.Link
	lesson.Conspect = input.Conspect

	err = h.Models.Lessons.Update(lesson)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"lesson": lesson})
}

func (h *LessonsHandler) DeleteLessonHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	err = h.Models.Lessons.Delete(id)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
