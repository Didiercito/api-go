package rabbitmq

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
	usuarios "go-api/src/user/model"
)

func PublishSearchRequest(req usuarios.SearchRequest) error {
	conn, err := amqp.Dial("amqp://Margarita:Didi@localhost:5672/")
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"usuarios_search_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	return err
}

func ConsumeSearchResults(searchID string, resultsChan chan<- []usuarios.Usuario) error {
	conn, err := amqp.Dial("amqp://Margarita:Didi@localhost:5672/")
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	queueName := "search_results_" + searchID

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		conn.Close()
		return err
	}

	msgs, err := ch.Consume(
		queueName,
		"",
		true, // auto-ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		conn.Close()
		return err
	}

	go func() {
		for d := range msgs {
			var users []usuarios.Usuario
			if err := json.Unmarshal(d.Body, &users); err != nil {
				log.Printf("Error decodificando resultados: %v", err)
				continue
			}
			resultsChan <- users
			break // Una vez recibido el primer mensaje con resultados, salimos
		}
		ch.Close()
		conn.Close()
	}()

	return nil
}

