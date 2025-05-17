# Sistema de Temperatura por CEP com OpenTelemetry e Zipkin

## Objetivo
Desenvolver um sistema em Go que recebe um CEP, identifica a cidade e retorna o clima atual (temperatura em graus Celsius, Fahrenheit e Kelvin) juntamente com a cidade. O sistema implementa tracing distribuído com OTEL (OpenTelemetry) e Zipkin.

## Estrutura do Projeto
- **service-a**: Responsável por receber o input do usuário, validar o CEP e encaminhar para o service-b.
- **service-b**: Responsável por orquestrar a busca da cidade (ViaCEP) e da temperatura (WeatherAPI), retornando o resultado formatado.
- **Zipkin**: Coletor de traces para visualização do tracing distribuído.

## Como rodar o projeto

### Pré-requisitos
- Docker e Docker Compose instalados
- Chave da WeatherAPI (https://www.weatherapi.com/)

### Passos
1. **Configure sua API Key**
   - Crie um arquivo `.env` na raiz do projeto com o conteúdo:
     ```
     WEATHER_API_KEY=sua_api_key_aqui
     TEST_MODE=false
     ```

2. **Suba os serviços**
   ```bash
   docker-compose up --build
   ```
   Isso irá iniciar:
   - service-a na porta 8081
   - service-b na porta 8082
   - zipkin na porta 9411

3. **Testando a API**
   - CEP válido:
     ```bash
     curl -X POST http://localhost:8081 -H "Content-Type: application/json" -d '{"cep":"29902555"}'
     ```
     Resposta esperada:
     ```json
     {"city":"Linhares","temp_C":27.6,"temp_F":81.68,"temp_K":300.6}
     ```
   - CEP inválido:
     ```bash
     curl -X POST http://localhost:8081 -H "Content-Type: application/json" -d '{"cep":"123"}'
     ```
     Resposta esperada: `invalid zipcode`
   - CEP não encontrado:
     ```bash
     curl -X POST http://localhost:8081 -H "Content-Type: application/json" -d '{"cep":"99999999"}'
     ```
     Resposta esperada: `can not find zipcode`

4. **Visualizando o tracing**
   - Acesse [http://localhost:9411](http://localhost:9411) no navegador
   - Você verá os traces de cada requisição, com todos os spans detalhados (validação, chamada de serviço, busca de cidade, busca de temperatura)
   - Exemplo de fluxo no Zipkin:
     - service-a: handle-cep-request
       - service-a: call-service-b
         - service-b: handle-weather-request
           - service-b: get-city-by-cep
           - service-b: get-temperature

## Requisitos atendidos
- [x] Recebe input via POST com schema `{ "cep": "29902555" }`
- [x] Valida se o input é uma string de 8 dígitos
- [x] Encaminha para o Serviço B via HTTP se válido
- [x] Retorna 422 e mensagem "invalid zipcode" se inválido
- [x] Consulta ViaCEP para obter a cidade
- [x] Consulta WeatherAPI para obter temperatura
- [x] Retorna JSON com cidade, temp_C, temp_F, temp_K
- [x] Retorna 404 e "can not find zipcode" se CEP não encontrado
- [x] Tracing distribuído OTEL + Zipkin implementado
- [x] Spans para cada etapa importante
- [x] Docker e Docker Compose funcionando

## Observações
- O projeto está pronto para ser entregue e testado.
- O tracing distribuído pode ser visualizado no Zipkin, mostrando toda a jornada da requisição.
- Para dúvidas ou melhorias, consulte os comentários no código ou abra uma issue.

---

**Parabéns! Seu sistema está moderno, rastreável e pronto para produção/teste acadêmico!**

# API de Clima por CEP

Esta API recebe um CEP brasileiro e retorna a temperatura atual da localidade em Celsius, Fahrenheit e Kelvin.

## Requisitos

- Go 1.21 ou superior
- Docker e Docker Compose
- Chave de API do WeatherAPI (https://www.weatherapi.com/)

## Configuração

1. Clone o repositório
2. Copie o arquivo `.env.example` para `.env`
3. Adicione sua chave de API do WeatherAPI no arquivo `.env`

## Executando localmente

```bash
go run main.go
```

## Executando com Docker

```bash
docker-compose up --build
```

## Uso

Faça uma requisição GET para a API com o CEP como parâmetro:

```
GET http://localhost:8080/?cep=12345678
```

### Respostas

#### Sucesso (200)
```json
{
    "temp_C": 28.5,
    "temp_F": 83.3,
    "temp_K": 301.5
}
```

#### CEP Inválido (422)
```
invalid zipcode
```

#### CEP Não Encontrado (404)
```
can not find zipcode
```

## Health Check

O endpoint de verificação de saúde pode ser acessado em:

```
GET http://localhost:8080/health
```

Resposta esperada:
```json
{
    "status": "ok"
}
```

## Testes

Para executar a aplicação em modo de teste (sem chamar APIs externas):

```bash
# Via variável de ambiente
TEST_MODE=true go run main.go

# Ou com Docker
docker-compose -f docker-compose.test.yml up --build
```

