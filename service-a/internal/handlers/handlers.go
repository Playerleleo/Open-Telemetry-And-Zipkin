package handlers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"service-a/internal/client"
	"service-a/internal/models"

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
			attribute.String("service.name", "service-a"),
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
	tracer = tracerProvider.Tracer("service-a-handlers")

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

// HandleCEPRequest processa requisições de CEP e encaminha para o Serviço B
func HandleCEPRequest(serviceBClient *client.ServiceBClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "handle-cep-request")
		defer span.End()

		// Verificar se é um POST
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
		var request models.CEPRequest
		if err := json.Unmarshal(body, &request); err != nil {
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		// Validar o CEP
		cep := request.CEP
		span.SetAttributes(attribute.String("cep", cep))

		// Verificar se o CEP contém exatamente 8 dígitos
		validCEP := regexp.MustCompile(`^\d{8}$`)
		if !validCEP.MatchString(cep) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("invalid zipcode"))
			return
		}

		// Enviar para o Serviço B
		weatherResponse, statusCode, err := serviceBClient.SendCEP(ctx, cep)
		if err != nil {
			if statusCode == http.StatusNotFound {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("can not find zipcode"))
				return
			} else if statusCode == http.StatusUnprocessableEntity {
				w.WriteHeader(http.StatusUnprocessableEntity)
				w.Write([]byte("invalid zipcode"))
				return
			} else {
				log.Printf("Erro ao chamar o Serviço B: %v", err)
				http.Error(w, "Error calling Service B", http.StatusInternalServerError)
				return
			}
		}

		// Retornar a resposta
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(weatherResponse)
	}
}
