package helpers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func BadRequestResponse(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func ServerErrorResponse(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func NotFoundResponse(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
}

func WriteJSON(c *gin.Context, statusCode int, data gin.H) {
	c.JSON(statusCode, data)
}

func ReadIDParam(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
