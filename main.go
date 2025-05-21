package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"go-api/src/db"
	"go-api/src/user/routes"
	"go-api/src/rabbitmq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error al cargar el archivo .env: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	database := db.GetDB()
	defer database.Close()

	r := mux.NewRouter()

	r.Use(middlewareCORS)
	go rabbitmq.StartUsuarioWorker()

	api := r.PathPrefix("/api/v1").Subrouter()
	routes.RegistrarUsuarioRoutes(api)

	fmt.Println("Servidor iniciado en el puerto", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}
