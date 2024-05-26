package handlers

import (
	"github.com/gin-gonic/gin"
	"lms-crud-api/internal/data"
	"lms-crud-api/internal/helpers"
	"net/http"
)

type ModulesHandler struct {
	Models data.Models
}

func (h *ModulesHandler) CreateModuleHandler(c *gin.Context) {
	var input struct {
		Title    string `json:"title"`
		CourseID uint   `json:"course_id"`
	}

	err := c.BindJSON(&input)
	if err != nil {
		helpers.BadRequestResponse(c, err)
		return
	}

	module := &data.Module{
		Title:    input.Title,
		CourseID: input.CourseID,
	}

	err = h.Models.Modules.Insert(module)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusCreated, gin.H{"module": module})
}

func (h *ModulesHandler) ShowAllModulesHandler(c *gin.Context) {
	modules, err := h.Models.Modules.GetAll() // Implement GetAll method in Modules model
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"modules": modules})
}

func (h *ModulesHandler) ShowModulesForCourseHandler(c *gin.Context) {
	courseID, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	modules, err := h.Models.Modules.GetAllWithLessonsForCourse(courseID) // Update GetAllWithLessonsForCourse to preload lessons
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"modules": modules})
}

func (h *ModulesHandler) ShowModuleHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	module, err := h.Models.Modules.GetWithLessons(id) // Update GetWithLessons to preload lessons
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"module": module})
}

func (h *ModulesHandler) UpdateModuleHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	var input struct {
		Title string `json:"title"`
	}

	err = c.BindJSON(&input)
	if err != nil {
		helpers.BadRequestResponse(c, err)
		return
	}

	module, err := h.Models.Modules.Get(id)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	module.Title = input.Title

	err = h.Models.Modules.Update(module)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	helpers.WriteJSON(c, http.StatusOK, gin.H{"module": module})
}

func (h *ModulesHandler) DeleteModuleHandler(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		helpers.NotFoundResponse(c)
		return
	}

	err = h.Models.Modules.Delete(id)
	if err != nil {
		helpers.ServerErrorResponse(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
