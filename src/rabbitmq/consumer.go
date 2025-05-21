package rabbitmq

import (
	"encoding/json"
	"log"
	"go-api/src/db"
	usuarios "go-api/src/user/model"

	"github.com/streadway/amqp"
)

func StartUsuarioWorker() {
	conn, err := amqp.Dial("amqp://Didi:Margarita@localhost:5672/")
	if err != nil {
		log.Fatalf("❌ Error conectando a RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("❌ Error abriendo canal: %v", err)
	}

	q, err := ch.QueueDeclare(
		"usuarios",
		true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("❌ Error declarando cola: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false, 
		false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("❌ Error al consumir mensajes: %v", err)
	}

	const workers = 10
	for i := 0; i < workers; i++ {
		go func(id int) {
			log.Printf("🚀 Worker %d iniciado...", id)
			for d := range msgs {
				var u usuarios.Usuario
				if err := json.Unmarshal(d.Body, &u); err != nil {
					log.Printf("⚠️ Worker %d: error al decodificar usuario: %v", id, err)
					d.Nack(false, false)
					continue
				}

				conn := db.GetDB()
				_, err := conn.Exec("INSERT INTO usuarios (CLVE_CLIENTE, NOMBRE_COMPLETO, CELULAR, EMAIL) VALUES ($1, $2, $3, $4)",
					u.CLVE_CLIENTE, u.NOMBRE_COMPLETO, u.CELULAR, u.EMAIL)
				if err != nil {
					log.Printf("❌ Worker %d: error al insertar usuario: %v", id, err)
					d.Nack(false, true) 
				} else {
					log.Printf("✅ Worker %d: usuario insertado: %v", id, u.CLVE_CLIENTE)
					d.Ack(false)
				}
			}
		}(i + 1)
	}

	log.Println("👷‍♂️ Todos los workers están ejecutándose...")
	select {} 
}

