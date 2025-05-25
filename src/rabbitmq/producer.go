package rabbitmq

import (
	"encoding/json"

	"github.com/streadway/amqp"
	usuarios "go-api/src/user/model"
)

// PublishUsuariosChunk recibe un slice de usuarios
func PublishUsuariosChunk(users []usuarios.Usuario) error {
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
		"usuarios_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	body, err := json.Marshal(users)
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