package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/auth"
	"github.com/ollama/ollama/auth/platform"
)

// writeUnauthorized returns the 401 used by /api/me. A signin_url pointing at
// the device flow is included whenever the link can be generated.
func writeUnauthorized(c *gin.Context) {
	response := gin.H{"error": "unauthorized"}

	if sURL, err := signinURL(); err == nil {
		response["signin_url"] = sURL
	}

	c.JSON(http.StatusUnauthorized, response)
}

// WhoamiHandler handles POST /api/me. It returns the locally stored Susan
// platform account, automatically refreshing expired access tokens.
func (s *Server) WhoamiHandler(c *gin.Context) {
	cred, err := platform.LoadCredentials()
	if errors.Is(err, platform.ErrCloudHostMismatch) {
		cred = nil
		err = nil
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	}

	// No credentials: return 401 without ever calling the platform profile.
	if cred == nil {
		writeUnauthorized(c)
		return
	}

	client, err := platform.DefaultClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	}

	session, err := platform.NewSession(client)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	}
	if !session.SignedIn() {
		// Credentials were removed between loading and session creation.
		writeUnauthorized(c)
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, client.BaseURL+"/api/user/profile", nil)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	}

	resp, err := session.Do(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, platform.ErrNotSignedIn) {
			writeUnauthorized(c)
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		writeUnauthorized(c)
		return
	case resp.StatusCode/100 == 5:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	case resp.StatusCode/100 != 2:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	}

	var profile platform.Profile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	}

	id, err := uuid.Parse(profile.ID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
		return
	}

	c.JSON(http.StatusOK, api.UserResponse{
		ID:    id,
		Email: profile.Email,
		Name:  profile.Username,
		Plan:  "free",
	})
}

// SignoutHandler handles POST /api/signout. It deletes the locally stored
// credentials; no remote disconnect call is made. With {"revoke": true} it
// first removes the current device from the user's platform account.
func (s *Server) SignoutHandler(c *gin.Context) {
	var req struct {
		Revoke bool `json:"revoke"`
	}
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	cred, err := platform.LoadCredentials()
	if errors.Is(err, platform.ErrCloudHostMismatch) {
		cred = nil
		err = nil
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "there was an error signing out"})
		return
	}

	if cred == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "you are not currently signed in"})
		return
	}

	if req.Revoke {
		client, err := platform.DefaultClient()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
			return
		}

		session, err := platform.NewSession(client)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "account unavailable"})
			return
		}

		publicKey, err := auth.GetPublicKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "there was an error signing out"})
			return
		}

		if err := session.RevokeCurrentDevice(c.Request.Context(), publicKey); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("could not revoke device: %v", err)})
			return
		}
	}

	if err := platform.DeleteCredentials(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "there was an error signing out"})
		return
	}

	// Cancel any in-progress device flow so it cannot rewrite credentials.
	platform.DefaultManager().Reset()

	c.JSON(http.StatusOK, gin.H{"status": "signed out"})
}
