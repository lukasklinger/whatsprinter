package main

import (
	"context"
	"fmt"
	"log"
	"mime"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
)

var client *whatsmeow.Client

func main() {
	connectWhatsApp()

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

// convert & print image from channel
func printImage(path string) {
	fmt.Println("Printing ", path)
}

// connect to whatsapp, set up event listeners
func connectWhatsApp() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)

	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("QR code:", evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}
}

// handle whatsapp events
func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
		extractImage(v.Message, v.Info.ID)
	}
}

// try to extract image from message
func extractImage(message *waE2E.Message, ID types.MessageID) {
	if img := message.GetImageMessage(); img != nil {
		imageData, err := client.Download(img)

		if err == nil {
			exts, _ := mime.ExtensionsByType(img.GetMimetype())
			fmt.Println(exts)
			path := fmt.Sprintf("./downloads/%s-%s%s", fmt.Sprint(time.Now().Unix()), ID, exts[0])

			err = os.WriteFile(path, imageData, 0600)
			if err != nil {
				log.Printf("Failed to save image: %v", err)
				return
			}

			log.Printf("Saved image in message to %s", path)
			printImage(path)
		} else {
			fmt.Println(err)
		}
	}
}
