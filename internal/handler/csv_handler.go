package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/felipemacedo1/transaction-producer-ms/internal/dto"
	"github.com/streadway/amqp"
)

func ProcessCSVFile(filePath string) error {

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return fmt.Errorf("rabbitmq connection could not be established: %v", err)
	}
	defer conn.Close()

	// create a channel
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel could not be created: %v", err)
	}
	defer ch.Close()

	queue, err := ch.QueueDeclare(
		"transaction_queue", // queue name
		true,                // durable
		false,               // auto-delete
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return fmt.Errorf("queue could not be declared: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("csv could not be read: %v", err)
		}

		age, err := strconv.Atoi(record[4])
		if err != nil {
			log.Printf("age could not be converted: %v", err)
			continue
		}

		amount, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			log.Printf("amount could not be converted: %v", err)
			continue
		}

		installments, err := strconv.Atoi(record[6])
		if err != nil {
			log.Printf("installments could not be converted: %v", err)
			continue
		}

		date, err := time.Parse(time.RFC3339, record[1])
		if err != nil {
			log.Printf("date could not be converted: %v", err)
			continue
		}

		transaction := dto.Transaction{
			TransactionID: record[0],
			Date:          date,
			ClientID:      record[2],
			Name:          record[3],
			Age:           age,
			Amount:        amount,
			Installments:  installments,
		}

		jsonData, err := json.Marshal(transaction)
		if err != nil {
			log.Printf("JSON could not be created: %v", err)
			continue
		}

		// publish the message to the queue
		err = ch.Publish(
			"",         // exchange
			queue.Name, // routing key
			false,      // mandatory
			false,      // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        jsonData,
			},
		)
		if err != nil {
			log.Printf("Message could not be sent: %v", err)
		}
	}

	return nil
}
