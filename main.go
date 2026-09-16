package main

import (
	"log"

	git "github.com/Xelckis/gitchat/internal/gitLogic"
	"github.com/Xelckis/gitchat/internal/web"
	"github.com/gin-gonic/gin"
)

func main() {

	hub := web.NewHub()
	go hub.Run()
	hubBroadcast := hub.BroadcastMsg

	git.StartSyncLoop("./repoDB")

	err := git.WatchRepo("./repoDB", hubBroadcast)
	if err != nil {
		log.Fatalf("Failed to start Git monitoring: %v", err)
	}

	user, err := git.SetUserConfig()
	if err != nil {
		panic(err)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	})

	router.StaticFile("/", "./index.html")

	router.POST("/message", web.Messages)
	router.GET("/history", web.GetHistory)
	router.GET("/ws", func(c *gin.Context) {
		web.ServeWs(hub, c.Writer, c.Request)
	})

	router.Run(":8081")

}
