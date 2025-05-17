package main

import (
	"log"
	"net/http"
	"os"

	"service-a/internal/client"
	"service-a/internal/handlers"
)

func main() {
	// Inicializar o cliente para o Serviço B
	serviceBURL := os.Getenv("SERVICE_B_URL")
	if serviceBURL == "" {
		serviceBURL = "http://service-b:8082"
	}

	// Inicializar o cliente de Serviço B
	serviceBClient := client.NewServiceBClient(serviceBURL)

	// Inicializar o tracer
	cleanupFunc := handlers.InitTracer()
	defer cleanupFunc()

	// Configurar rotas
	http.HandleFunc("/", handlers.HandleCEPRequest(serviceBClient))
	http.HandleFunc("/health", handlers.HandleHealthCheck)

	// Definir porta
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Serviço A iniciado na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
