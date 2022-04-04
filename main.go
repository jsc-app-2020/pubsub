package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

var sessions = make(map[string]chan string)

func socketHandler(topic string, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	closed := make(chan bool)
	defer conn.Close()
	defer close(closed)

	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				closed <- true
				break
			}
		}
	}()

	session := topic
	channel := make(chan string)
	sessions[session] = channel

	defer delete(sessions, session)
	defer close(channel)

	// listening channel
	listening := true
	for listening {
		select {
		case <-closed:
			listening = false

		case message := <-channel:
			conn.WriteMessage(websocket.TextMessage, []byte(message))
		}
	}

	log.Println("Disconnected")
}

func pubHandler(c *gin.Context) {
	topic := c.Param("topic")

	bytes, err := c.GetRawData()
	if err != nil {
		c.Status(400)
		return
	}

	if session, ok := sessions[topic]; ok {
		session <- string(bytes)
		c.Status(200)
		return
	}

	c.Status(404)
}

func main() {
	if _, err := os.Stat(".env"); !errors.Is(err, os.ErrNotExist) {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	r.POST("/pub/:topic", pubHandler)
	r.GET("sub/:topic", func(ctx *gin.Context) {
		topic := ctx.Param("topic")
		socketHandler(topic, ctx.Writer, ctx.Request)
	})

	port := os.Getenv("PORT")
	log.Fatal(r.Run(":" + port))
}
