package rabbitmq

import (
	"database/sql"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"go-api/src/db"
	usuarios "go-api/src/user/model"

	"github.com/streadway/amqp"
)

func StartUsuarioWorker() {
	conn, err := amqp.Dial("amqp://Margarita:Didi@localhost:5672/")
	if err != nil {
		log.Fatalf("❌ Error conectando a RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("❌ Error abriendo canal: %v", err)
	}
	defer ch.Close()

	err = ch.Qos(
		100,  
		0,    
		false,
	)
	if err != nil {
		log.Fatalf("❌ Error configurando QoS: %v", err)
	}

	q, err := ch.QueueDeclare(
		"usuarios_queue",
		true, 
		false, 
		false, 
		false, 
		nil,
	)
	if err != nil {
		log.Fatalf("❌ Error declarando cola: %v", err)
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
		log.Fatalf("❌ Error al consumir mensajes: %v", err)
	}

	const workers = 12
	for i := 0; i < workers; i++ {
		go func(id int) {
			log.Printf("🚀 Worker %d iniciado...", id)

			dbConn := db.GetDB()

			for d := range msgs {
				var users []usuarios.Usuario
				if err := json.Unmarshal(d.Body, &users); err != nil {
					log.Printf("⚠️ Worker %d: error al decodificar lote de usuarios: %v", id, err)
					d.Nack(false, false) 
					continue
				}

				for _, u := range users {
					logDirtyData(id, u)
				}

				if err := insertOrUpdateUsersBatch(dbConn, users, 3); err != nil {
					log.Printf("❌ Worker %d: error final al procesar lote: %v", id, err)
					d.Nack(false, true)
				} else {
					log.Printf("✅ Worker %d: lote de %d usuarios procesado correctamente", id, len(users))
					d.Ack(false)
				}
			}
		}(i + 1)
	}

	log.Printf("👷‍♂️ %d workers ejecutándose para procesar usuarios...", workers)
	log.Printf("📝 Guardando datos SIN limpiar - frontend se encargará de correcciones")
	select {} 
}

func logDirtyData(workerID int, u usuarios.Usuario) {
	var issues []string

	if strings.ContainsAny(u.NOMBRE_COMPLETO, "0123456789!@#$%^&*()") {
		issues = append(issues, "nombre_con_caracteres_especiales")
	}

	if len(u.CELULAR) < 10 || strings.ContainsAny(u.CELULAR, "abcdefghijklmnopqrstuvwxyz") {
		issues = append(issues, "celular_formato_irregular")
	}

	if !strings.Contains(u.EMAIL, "@") || !strings.Contains(u.EMAIL, ".") {
		issues = append(issues, "email_formato_irregular")
	}

	if len(issues) > 0 {
		log.Printf("📝 Worker %d: Usuario %d tiene datos para revisar: %v",
			workerID, u.CLVE_CLIENTE, issues)
	}
}

func insertOrUpdateUsersBatch(db *sql.DB, users []usuarios.Usuario, maxRetries int) error {
	if len(users) == 0 {
		return nil
	}

	var err error

	for i := 0; i < maxRetries; i++ {
		tx, txErr := db.Begin()
		if txErr != nil {
			err = txErr
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			continue
		}

		valueStrings := make([]string, 0, len(users))
		valueArgs := make([]interface{}, 0, len(users)*4)

		for i, u := range users {
			startPos := i*4 + 1
			valueStrings = append(valueStrings, 
				"($" + 
				strconv.Itoa(startPos) + "," + 
				"$" + strconv.Itoa(startPos+1) + "," + 
				"$" + strconv.Itoa(startPos+2) + "," + 
				"$" + strconv.Itoa(startPos+3) + ")",
			)
			valueArgs = append(valueArgs, u.CLVE_CLIENTE, u.NOMBRE_COMPLETO, u.CELULAR, u.EMAIL)
		}

		query := `
			INSERT INTO usuarios (CLVE_CLIENTE, NOMBRE_COMPLETO, CELULAR, EMAIL) VALUES 
			` + strings.Join(valueStrings, ",") + `
			ON CONFLICT (CLVE_CLIENTE) DO UPDATE SET 
				NOMBRE_COMPLETO = EXCLUDED.NOMBRE_COMPLETO,
				CELULAR = EXCLUDED.CELULAR,
				EMAIL = EXCLUDED.EMAIL
		`

		_, err = tx.Exec(query, valueArgs...)
		if err != nil {
			tx.Rollback()
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			continue
		}

		err = tx.Commit()
		if err == nil {
			return nil
		}

		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}

	return err
}