package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

type Message struct {
	ZoneID int    `json:"zone_id"`
	Type   string `json:"type"`
}

type Zone struct {
	ID                int
	RemainingCapacity int
}

func main() {
	// Configure Viper to read from environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("MYAPP")

	// Read the queue name from Viper
	queueName := viper.GetString("RABBITMQ_QUEUE")
	rabbitURL := viper.GetString("RABBITMQ_URL")
	fmt.Println(queueName)

	// Reconnection loop
	for {
		// Attempt to connect to RabbitMQ
		conn, err := amqp.Dial(rabbitURL)
		if err != nil {
			log.Println("Failed to connect to RabbitMQ:", err)
			// Add a delay before attempting to reconnect
			time.Sleep(5 * time.Second)
			continue
		}
		defer conn.Close()

		channel, err := conn.Channel()
		if err != nil {
			log.Println("Failed to open channel:", err)
			// Add a delay before attempting to reconnect
			time.Sleep(5 * time.Second)
			continue
		}
		defer channel.Close()

		// Attempt to consume messages from the queue
		messages, err := channel.Consume(
			queueName,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Println("Failed to consume messages:", err)
			// Add a delay before attempting to reconnect
			time.Sleep(5 * time.Second)
			continue
		}

		// Read database connection details from Viper
		dbUsername := viper.GetString("DB_USERNAME")
		dbPassword := viper.GetString("DB_PASSWORD")
		dbName := viper.GetString("DB_NAME")
		dbHost := viper.GetString("DB_HOST")
		dbPort := viper.GetString("DB_PORT")
		dbSSLMode := viper.GetString("DB_SSLMODE")

		// Construct the database connection URL
		dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			dbUsername, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
		db, err := gorm.Open("postgres", dbURL)
		if err != nil {
			log.Println("Failed to connect to the database:", err)
			// Add a delay before attempting to reconnect
			time.Sleep(5 * time.Second)
			continue
		}
		defer db.Close()

		for msg := range messages {
			var message Message
			err := json.Unmarshal(msg.Body, &message)
			if err != nil {
				log.Println("Failed to unmarshal message:", err)
				continue
			}

			// Perform the database query based on the message type
			var zone Zone
			db.First(&zone, message.ZoneID)

			if message.Type == "enter" {
				zone.RemainingCapacity--
			} else if message.Type == "exit" {
				zone.RemainingCapacity++
			}

			db.Save(&zone)
			fmt.Printf("Updated zone %d - Remaining capacity: %d\n", zone.ID, zone.RemainingCapacity)
		}
	}
}
