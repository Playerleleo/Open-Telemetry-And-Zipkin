package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	viaCEPURL     = "https://viacep.com.br/ws/%s/json/"
	weatherAPIURL = "http://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no"
	testMode      = false // Modo de teste desativado por padrão
	tracer        trace.Tracer
)

type RequestPayload struct {
	CEP string `json:"cep"`
}

type WeatherResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

type ViaCEPResponse struct {
	Cidade string `json:"localidade"`
}

type WeatherAPIResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
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
			attribute.String("service.name", "service-b"),
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

	// Configurar o propagador para receber contexto entre serviços
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)

	tracer = tracerProvider.Tracer("service-b")
}

func main() {
	// Inicializar tracer
	initTracer()

	// Verificar se estamos em modo de teste a partir de variável de ambiente
	testModeEnv := os.Getenv("TEST_MODE")
	if testModeEnv == "true" {
		testMode = true
		log.Println("Iniciando em modo de teste")
	}

	// Configurar rotas
	http.HandleFunc("/", handleWeatherRequest)
	http.HandleFunc("/health", handleHealthCheck)

	// Configurar porta
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Serviço B iniciado na porta %s", port)
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

func handleWeatherRequest(w http.ResponseWriter, r *http.Request) {
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
	var payload RequestPayload
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
	cidade, err := getCityByCEP(ctx, cep)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("can not find zipcode"))
		return
	}

	// Buscar temperatura
	tempC, err := getTemperature(ctx, cidade)
	if err != nil {
		log.Printf("Erro ao obter temperatura: %v", err)
		http.Error(w, "Error getting temperature", http.StatusInternalServerError)
		return
	}

	// Converter temperaturas
	tempF := tempC*1.8 + 32
	tempK := tempC + 273

	response := WeatherResponse{
		City:  cidade,
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCityByCEP(ctx context.Context, cep string) (string, error) {
	ctx, span := tracer.Start(ctx, "get-city-by-cep")
	defer span.End()

	span.SetAttributes(attribute.String("cep", cep))

	// Para testes: simular CEP não encontrado
	if os.Getenv("SIMULATE_CEP_NOT_FOUND") == "true" {
		return "", fmt.Errorf("CEP not found")
	}

	url := fmt.Sprintf(viaCEPURL, cep)

	log.Printf("Consultando CEP: %s", url)

	// Criar requisição com contexto
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	// Executar requisição
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Erro ao consultar ViaCEP: %v", err)
		span.RecordError(err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("ViaCEP retornou status code: %d", resp.StatusCode)
		err := fmt.Errorf("CEP not found")
		span.RecordError(err)
		return "", err
	}

	var viaCEPResp ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&viaCEPResp); err != nil {
		log.Printf("Erro ao decodificar resposta do ViaCEP: %v", err)
		span.RecordError(err)
		return "", err
	}

	if viaCEPResp.Cidade == "" {
		err := fmt.Errorf("CEP not found")
		span.RecordError(err)
		return "", err
	}

	log.Printf("Cidade encontrada: %s", viaCEPResp.Cidade)
	span.SetAttributes(attribute.String("city", viaCEPResp.Cidade))
	return viaCEPResp.Cidade, nil
}

func getTemperature(ctx context.Context, cidade string) (float64, error) {
	ctx, span := tracer.Start(ctx, "get-temperature")
	defer span.End()

	span.SetAttributes(attribute.String("city", cidade))

	// Modo de teste retorna valor fictício para facilitar testes
	if testMode {
		log.Printf("Usando modo de teste para cidade: %s", cidade)
		return 25.0, nil
	}

	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("WEATHER_API_KEY not set")
	}

	// Normaliza a string removendo acentos
	encodedCidade := removeAccents(cidade)

	url := fmt.Sprintf(weatherAPIURL, apiKey, encodedCidade)

	log.Printf("Consultando temperatura para %s: %s", cidade, url)

	// Criar requisição com contexto
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	// Executar requisição
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Erro ao consultar API: %v", err)
		span.RecordError(err)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API retornou status code: %d", resp.StatusCode)
		err := fmt.Errorf("Error getting weather data: status %d", resp.StatusCode)
		span.RecordError(err)
		return 0, err
	}

	var weatherResp WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		log.Printf("Erro ao decodificar resposta: %v", err)
		span.RecordError(err)
		return 0, err
	}

	// Registrar a temperatura encontrada
	span.SetAttributes(attribute.Float64("temperature_c", weatherResp.Current.TempC))
	return weatherResp.Current.TempC, nil
}

// Função para remover acentos de uma string
func removeAccents(s string) string {
	replacements := map[string]string{
		"á": "a", "à": "a", "ã": "a", "â": "a", "ä": "a",
		"é": "e", "è": "e", "ê": "e", "ë": "e",
		"í": "i", "ì": "i", "î": "i", "ï": "i",
		"ó": "o", "ò": "o", "õ": "o", "ô": "o", "ö": "o",
		"ú": "u", "ù": "u", "û": "u", "ü": "u",
		"ç": "c",
		"Á": "A", "À": "A", "Ã": "A", "Â": "A", "Ä": "A",
		"É": "E", "È": "E", "Ê": "E", "Ë": "E",
		"Í": "I", "Ì": "I", "Î": "I", "Ï": "I",
		"Ó": "O", "Ò": "O", "Õ": "O", "Ô": "O", "Ö": "O",
		"Ú": "U", "Ù": "U", "Û": "U", "Ü": "U",
		"Ç": "C",
	}

	result := s
	for from, to := range replacements {
		result = strings.Replace(result, from, to, -1)
	}
	return result
}
