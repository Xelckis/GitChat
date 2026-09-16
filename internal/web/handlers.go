package web

import (
	"log"
	"net/http"
	"strconv"

	git "github.com/Xelckis/gitchat/internal/gitLogic"
	"github.com/gin-gonic/gin"
)

func sendJSON(c *gin.Context, code int, status, message, nick string) {
	c.JSON(code, gin.H{"status": status,
		"error":   message,
		"message": "",
		"nick":    nick,
	})
}

func Messages(c *gin.Context) {
	msg := c.PostForm("message")
	if msg == "" {
		sendJSON(c, 400, "error", "Message is required", "")
		return
	}

	userConfig, ok := c.Get("user")
	if !ok {
		sendJSON(c, 400, "error", "User config not found", "")
		return
	}

	user := userConfig.(git.User)

	err := git.SendMessage(msg, user.Name, user.Email)
	if err != nil {
		log.Println(err)
		sendJSON(c, 500, "error", "Failed to send message", "")
		return
	}

}

func GetHistory(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit := 50
	offset := (page - 1) * limit

	commits, err := git.ReadHistory(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load history"})
		return
	}

	c.JSON(http.StatusOK, commits)
}
