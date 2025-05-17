package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"service-a/internal/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ServiceBClient é responsável pela comunicação com o Serviço B
type ServiceBClient struct {
	baseURL string
	client  *http.Client
	tracer  trace.Tracer
}

// NewServiceBClient cria uma nova instância do cliente do Serviço B
func NewServiceBClient(baseURL string) *ServiceBClient {
	return &ServiceBClient{
		baseURL: baseURL,
		client:  &http.Client{},
		tracer:  otel.GetTracerProvider().Tracer("service-a-client"),
	}
}

// SendCEP envia um CEP para o Serviço B e retorna a resposta com temperatura
func (c *ServiceBClient) SendCEP(ctx context.Context, cep string) (*models.WeatherResponse, int, error) {
	ctx, span := c.tracer.Start(ctx, "call-service-b")
	defer span.End()

	span.SetAttributes(attribute.String("cep", cep))

	// Preparar a requisição para o Serviço B
	requestBody := models.ServiceBRequest{
		CEP: cep,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		span.RecordError(err)
		return nil, http.StatusInternalServerError, fmt.Errorf("error marshaling request: %w", err)
	}

	// Criar a requisição HTTP
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		span.RecordError(err)
		return nil, http.StatusInternalServerError, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Injetar o contexto de trace no cabeçalho da requisição
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// Enviar a requisição
	resp, err := c.client.Do(req)
	if err != nil {
		span.RecordError(err)
		return nil, http.StatusInternalServerError, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// Ler o corpo da resposta
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		return nil, resp.StatusCode, fmt.Errorf("error reading response: %w", err)
	}

	// Se o status não for de sucesso, retornar erro
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("service B returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Decodificar a resposta JSON
	var weatherResponse models.WeatherResponse
	if err := json.Unmarshal(respBody, &weatherResponse); err != nil {
		span.RecordError(err)
		return nil, http.StatusInternalServerError, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &weatherResponse, resp.StatusCode, nil
}
