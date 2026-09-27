package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/auth/platform"
	"github.com/ollama/ollama/envconfig"
)

type signinDeviceRequest struct {
	ClientName string `json:"client_name"`
	Force      bool   `json:"force"`
}

// StartSigninDeviceHandler starts (or reuses an in-progress) device flow. The
// daemon polls the Susan platform in the background.
func (s *Server) StartSigninDeviceHandler(c *gin.Context) {
	var req signinDeviceRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if !envconfig.CloudHostConfigured() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "Susan cloud is not configured; set the SUSAN_CLOUD_HOST environment variable to your Susan platform (for example http://localhost:8000) and restart the daemon",
		})
		return
	}

	status, err := platform.DefaultManager().Start(c.Request.Context(), req.ClientName, req.Force)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("could not reach Susan platform: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// SigninDeviceStatusHandler returns the current device flow status. Its body
// never contains the device code or tokens.
func (s *Server) SigninDeviceStatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, platform.DefaultManager().CurrentStatus())
}
