package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"log"
)

var connectedUsers []User

type User struct {
	id string
	ws *websocket.Conn
}

type Msg struct {
	Type      string `json:"type,omitempty"`
	SocketID  string `json:"socket_id,omitempty"`
	SDP       string `json:"sdp,omitempty"`
	Candidate string `json:"candidate,omitempty"`
}

type OtherUsers struct {
	Type       string   `json:"type,omitempty"`
	OtherUsers []string `json:"other_users"`
}

func main() {
	app := fiber.New()
	app.Get("/", func(context *fiber.Ctx) error {
		return context.SendFile("./index.html")
	})

	app.Static("/", "/")

	app.Use("/ws", func(context *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(context) {
			context.Locals("id", uuid.New().String())
			return context.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(ws *websocket.Conn) {
		connectedUsers = append(connectedUsers, User{
			id: ws.Locals("id").(string),
			ws: ws,
		})

		var connectedUsersIDs []string
		for i := range connectedUsers {
			if connectedUsers[i].id != ws.Locals("id").(string) {
				connectedUsersIDs = append(connectedUsersIDs, connectedUsers[i].id)
			}
		}

		if err := ws.WriteJSON(OtherUsers{
			Type:       "other-users",
			OtherUsers: connectedUsersIDs,
		}); err != nil {
			log.Fatalln("Send other users failed")
		}

		var msg Msg
		for {
			msg = Msg{}
			err := ws.ReadJSON(&msg)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway) {
					for i := range connectedUsers {
						if connectedUsers[i].id == ws.Locals("id").(string) {
							connectedUsers[i] = connectedUsers[len(connectedUsers)-1]
							connectedUsers = connectedUsers[:len(connectedUsers)-1]
							break
						}
					}
					//connectedUsers = append(connectedUsers[:i], connectedUsers[i+1:]...)
					_ = ws.Close()
					return
				} else {
					fmt.Println("ERROR HAPPENED :/")
					continue
				}
			}
			switch msg.Type {
			case "offer":
				for _, user := range connectedUsers {
					if user.id != ws.Locals("id").(string) {
						_ = user.ws.WriteJSON(Msg{
							Type:     "offer",
							SocketID: ws.Locals("id").(string),
							SDP:      msg.SDP,
						})
						break
					}
				}
			case "answer":
				for _, user := range connectedUsers {
					if user.id != ws.Locals("id").(string) {
						_ = user.ws.WriteJSON(Msg{
							Type: "answer",
							SDP:  msg.SDP,
						})
						break
					}
				}
			case "candidate":
				for _, user := range connectedUsers {
					if user.id != ws.Locals("id").(string) {
						_ = user.ws.WriteJSON(Msg{
							Type:      "candidate",
							Candidate: msg.Candidate,
						})
						break
					}
				}
			default:
				fmt.Println("INVALID MESSAGE TYPE")
				fmt.Println(err)
				continue
			}
		}
	}))

	log.Fatalln(app.Listen(":3000"))
}
