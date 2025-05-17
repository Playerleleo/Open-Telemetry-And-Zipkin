package handlers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"service-b/internal/models"
	"service-b/internal/services"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// InitTracer inicializa o tracer OpenTelemetry e retorna uma função para limpeza
func InitTracer() func() {
	// Endereço do Zipkin
	zipkinURL := "http://zipkin:9411/api/v2/spans"
	if os.Getenv("ZIPKIN_URL") != "" {
		zipkinURL = os.Getenv("ZIPKIN_URL")
	}

	// Criar exporter para Zipkin
	exporter, err := zipkin.New(zipkinURL)
	if err != nil {
		log.Fatalf("Erro ao criar exporter do Zipkin: %v", err)
	}

	// Criar resource que representa a aplicação
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			attribute.String("service.name", "service-b"),
			attribute.String("service.version", "0.1.0"),
		),
	)
	if err != nil {
		log.Fatalf("Erro ao criar resource: %v", err)
	}

	// Configurar o provider de tracer
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	// Configurar propagador
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)

	// Criar tracer
	tracer = tracerProvider.Tracer("service-b-handlers")

	// Retornar função para limpeza de recursos quando a aplicação for encerrada
	return func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			log.Printf("Erro ao encerrar tracer provider: %v", err)
		}
	}
}

// HandleHealthCheck verifica se o serviço está ativo
func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// HandleWeatherRequest processa as requisições de CEP e retorna os dados de temperatura
func HandleWeatherRequest(weatherService *services.WeatherService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extrair o contexto de propagação do cabeçalho da requisição
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(ctx, "handle-weather-request")
		defer span.End()

		// Aceita apenas método POST
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Ler o corpo da requisição
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Decodificar o JSON
		var payload models.CEPRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid request format"))
			return
		}

		// Obter o CEP do payload
		cep := payload.CEP

		// Validar formato do CEP
		validCEP := regexp.MustCompile(`^\d{8}$`)
		if !validCEP.MatchString(cep) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("invalid zipcode"))
			return
		}

		// Buscar cidade pelo CEP
		cidade, err := weatherService.GetCityByCEP(ctx, cep)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("can not find zipcode"))
			return
		}

		// Buscar temperatura
		tempC, err := weatherService.GetTemperature(ctx, cidade)
		if err != nil {
			log.Printf("Erro ao obter temperatura: %v", err)
			http.Error(w, "Error getting temperature", http.StatusInternalServerError)
			return
		}

		// Converter temperaturas
		tempF := tempC*1.8 + 32
		tempK := tempC + 273

		// Montar resposta
		response := models.WeatherResponse{
			City:  cidade,
			TempC: tempC,
			TempF: tempF,
			TempK: tempK,
		}

		// Enviar resposta
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
