package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

var sessions = make(map[string]chan string)

func generateSession() string {
	n := 16
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%X", b)
	return s
}

func socketHandler(id string, w http.ResponseWriter, r *http.Request) {
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

	session := id
	if session == "" {
		// generate session and store channel
		session = generateSession()
	}

	channel := make(chan string)
	sessions[session] = channel

	defer delete(sessions, session)
	defer close(channel)

	type msg struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
	sessionData := msg{
		Type: "session",
		Data: session,
	}

	err = conn.WriteJSON(sessionData)
	if err != nil {
		log.Print("Error during sending session:", err)
		return
	}

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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	r.POST("/pub/:topic", pubHandler)
	r.GET("/sub", func(c *gin.Context) {
		socketHandler("", c.Writer, c.Request)
	})

	r.GET("sub/:topic", func(ctx *gin.Context) {
		topic := ctx.Param("topic")
		socketHandler(topic, ctx.Writer, ctx.Request)
	})

	port := os.Getenv("PORT")
	log.Fatal(r.Run(":" + port))
}
