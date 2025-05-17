package main

import (
	"log"
	"net/http"
	"os"

	"service-b/internal/handlers"
	"service-b/internal/services"
)

func main() {
	// Inicializar o tracer
	cleanupFunc := handlers.InitTracer()
	defer cleanupFunc()

	// Inicializar serviços
	weatherService := services.NewWeatherService()

	// Configurar rotas
	http.HandleFunc("/", handlers.HandleWeatherRequest(weatherService))
	http.HandleFunc("/health", handlers.HandleHealthCheck)

	// Configurar porta
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Serviço B iniciado na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
