package util

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// SafeErrorResponse returns a JSON error response, logging details but only exposing safe info to users
func SafeErrorResponse(c *gin.Context, statusCode int, userMessage string, err error) {
	// Always log the detailed error for debugging
	if err != nil {
		log.Printf("[ERROR] %s: %v", c.Request.URL.Path, err)
	}

	response := gin.H{
		"success": false,
		"message": userMessage,
	}

	// Only include detailed error in development mode
	if os.Getenv("GIN_MODE") != "release" && err != nil {
		response["error"] = err.Error()
	}

	c.JSON(statusCode, response)
}
