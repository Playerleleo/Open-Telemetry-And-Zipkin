package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	serviceBURL = "http://service-b:8082/"
	tracer      trace.Tracer
)

// Estrutura para receber o CEP do cliente
type CEPRequest struct {
	CEP string `json:"cep"`
}

// Estrutura para repassar o CEP para o Serviço B
type ServiceBRequest struct {
	CEP string `json:"cep"`
}

func initTracer() {
	// Endereço do Zipkin (por padrão, o Zipkin escuta na porta 9411)
	zipkinURL := "http://zipkin:9411/api/v2/spans"
	if os.Getenv("ZIPKIN_URL") != "" {
		zipkinURL = os.Getenv("ZIPKIN_URL")
	}

	// Cria um exporter para Zipkin
	exporter, err := zipkin.New(zipkinURL)
	if err != nil {
		log.Fatalf("Erro ao criar exporter do Zipkin: %v", err)
	}

	// Cria um resource que representa a aplicação
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

	// Configura o tracer global
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	// Configurar o propagador para propagar contexto entre serviços
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)

	tracer = tracerProvider.Tracer("service-a")
}

func main() {
	// Inicializar tracer
	initTracer()

	// Configurar rotas
	http.HandleFunc("/", handleRequest)
	http.HandleFunc("/health", handleHealthCheck)

	// Definir porta
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Serviço A iniciado na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Endpoint para verificação de saúde (health check)
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
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
	var request CEPRequest
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

	// Preparar a requisição para o Serviço B
	serviceBRequest := ServiceBRequest{
		CEP: cep,
	}
	reqBody, err := json.Marshal(serviceBRequest)
	if err != nil {
		http.Error(w, "Error preparing request to Service B", http.StatusInternalServerError)
		return
	}

	// Enviar a requisição para o Serviço B
	resp, err := sendRequestToServiceB(ctx, reqBody)
	if err != nil {
		log.Printf("Erro ao chamar o Serviço B: %v", err)
		http.Error(w, fmt.Sprintf("Error calling Service B: %v", err), http.StatusInternalServerError)
		return
	}

	// Copiar a resposta do Serviço B para o cliente
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response from Service B", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	w.Write(responseBody)
}

func sendRequestToServiceB(ctx context.Context, reqBody []byte) (*http.Response, error) {
	ctx, span := tracer.Start(ctx, "call-service-b")
	defer span.End()

	// Preparar a requisição
	req, err := http.NewRequestWithContext(ctx, "POST", serviceBURL, bytes.NewBuffer(reqBody))
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Injetar o contexto de trace no cabeçalho da requisição
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// Enviar a requisição
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
	return resp, nil
}
