package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var isLocked bool = false

func GetLockStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"locked": isLocked})
}

func PutLockStatus(c *gin.Context) {
	var request struct {
		Locked bool `json:"locked"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, "Invalid JSON format")
		return
	}

	isLocked = request.Locked

	if isLocked {
		c.JSON(http.StatusOK, gin.H{"status": "locked"})
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "unlocked"})
	}
}