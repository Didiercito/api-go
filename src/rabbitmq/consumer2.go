package rabbitmq

import (
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"strconv"

	"go-api/src/db"
	usuarios "go-api/src/user/model"

	"github.com/streadway/amqp"
)

func StartSearchWorker() {
	conn, err := amqp.Dial("amqp://Margarita:Didi@localhost:5672/")
	if err != nil {
		log.Fatalf("❌ Error conectando a RabbitMQ: %v", err)
	}
	defer conn.Close()

	dbConn := db.GetDB()

	const workerCount = 12

	for i := 0; i < workerCount; i++ {
		go func(id int) {
			ch, err := conn.Channel()
			if err != nil {
				log.Fatalf("❌ Worker %d: error abriendo canal: %v", id, err)
			}
			defer ch.Close()

			err = ch.Qos(100, 0, false)
			if err != nil {
				log.Fatalf("❌ Worker %d: error configurando QoS: %v", id, err)
			}

			q, err := ch.QueueDeclare(
				"usuarios_search_queue",
				true,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				log.Fatalf("❌ Worker %d: error declarando cola: %v", id, err)
			}

			msgs, err := ch.Consume(
				q.Name,
				"",
				false,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				log.Fatalf("❌ Worker %d: error al consumir mensajes: %v", id, err)
			}

			log.Printf("🔎 Worker de búsqueda %d iniciado...", id)

			for d := range msgs {
				var req usuarios.SearchRequest
				if err := json.Unmarshal(d.Body, &req); err != nil {
					log.Printf("⚠️ Worker %d: error decodificando petición: %v", id, err)
					d.Nack(false, false)
					continue
				}

				results, err := searchUsers(dbConn, req)
				if err != nil {
					log.Printf("❌ Worker %d: error buscando usuarios: %v", id, err)
					d.Nack(false, true)
					continue
				}

				err = publishSearchResults(ch, req.SearchID, results)
				if err != nil {
					log.Printf("❌ Worker %d: error publicando resultados: %v", id, err)
					d.Nack(false, true)
					continue
				}

				log.Printf("✅ Worker %d: búsqueda con SearchID %s procesada, %d resultados", id, req.SearchID, len(results))
				d.Ack(false)
			}
		}(i + 1)
	}

	log.Printf("👷‍♂️ %d workers de búsqueda ejecutándose...", workerCount)
	select {}
}

func searchUsers(db *sql.DB, req usuarios.SearchRequest) ([]usuarios.Usuario, error) {
	conds := []string{}
	args := []interface{}{}
	argID := 1

	if req.CLVE_CLIENTE != 0 {
		conds = append(conds, "CLVE_CLIENTE = $"+strconv.Itoa(argID))
		args = append(args, req.CLVE_CLIENTE)
		argID++
	}
	if req.NOMBRE_COMPLETO != "" {
		conds = append(conds, "NOMBRE_COMPLETO ILIKE '%' || $"+strconv.Itoa(argID)+" || '%'")
		args = append(args, req.NOMBRE_COMPLETO)
		argID++
	}
	if req.CELULAR != "" {
		conds = append(conds, "CELULAR ILIKE '%' || $"+strconv.Itoa(argID)+" || '%'")
		args = append(args, req.CELULAR)
		argID++
	}
	if req.EMAIL != "" {
		conds = append(conds, "EMAIL ILIKE '%' || $"+strconv.Itoa(argID)+" || '%'")
		args = append(args, req.EMAIL)
		argID++
	}

	query := "SELECT CLVE_CLIENTE, NOMBRE_COMPLETO, CELULAR, EMAIL FROM usuarios"
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " LIMIT 2000"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []usuarios.Usuario{}
	for rows.Next() {
		var u usuarios.Usuario
		if err := rows.Scan(&u.CLVE_CLIENTE, &u.NOMBRE_COMPLETO, &u.CELULAR, &u.EMAIL); err != nil {
			return nil, err
		}
		results = append(results, u)
	}

	return results, nil
}

func publishSearchResults(ch *amqp.Channel, searchID string, users []usuarios.Usuario) error {
	queueName := "search_results_" + searchID

	_, err := ch.QueueDeclare(
		queueName,
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

	return ch.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
