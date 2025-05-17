# Guia de Deploy no Google Cloud Run

Este documento explica como implantar a API de Clima por CEP no Google Cloud Run.

## Pré-requisitos

1. Ter uma conta do Google Cloud Platform (GCP)
2. Ter o Google Cloud SDK instalado localmente
3. Ter uma chave da API do WeatherAPI (https://www.weatherapi.com/)

## Configurações de Ambiente Importantes

Para o funcionamento correto da API, é necessário configurar as seguintes variáveis de ambiente no Cloud Run:

```
WEATHER_API_KEY=sua_chave_api_aqui
```

Ou, para o modo de teste (sem chamar a API externa):

```
TEST_MODE=true
```

**Nota:** Se você não configurar a variável `WEATHER_API_KEY` ou não ativar o `TEST_MODE`, a API retornará erro ao tentar buscar a temperatura.

## Opção 1: Deploy manual

### 1. Autenticação no Google Cloud

```bash
gcloud auth login
```

### 2. Configurar o projeto GCP

```bash
gcloud config set project [SEU_PROJECT_ID]
```

### 3. Build da imagem Docker

```bash
docker build -t gcr.io/[SEU_PROJECT_ID]/cep-weather-api .
```

### 4. Configurar o Docker para o Google Cloud Registry

```bash
gcloud auth configure-docker
```

### 5. Push da imagem para o Container Registry

```bash
docker push gcr.io/[SEU_PROJECT_ID]/cep-weather-api
```

### 6. Deploy no Cloud Run

```bash
gcloud run deploy cep-weather-api \
  --image gcr.io/[SEU_PROJECT_ID]/cep-weather-api \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars="WEATHER_API_KEY=[SUA_CHAVE_API]"
```

## Opção 2: Deploy automatizado com Cloud Build

### 1. Ativar as APIs necessárias

- Cloud Build API
- Cloud Run API
- Container Registry API

### 2. Configurar uma Trigger no Cloud Build

1. Acesse o console do GCP e navegue até o Cloud Build
2. Em "Triggers", clique em "Create Trigger"
3. Configure a trigger para o seu repositório Git
4. Em "Configuration", selecione "Cloud Build configuration file (yaml or json)"
5. Certifique-se de que o caminho do arquivo seja `cloudbuild.yaml`

### 3. Configurar variáveis de substituição 

No console do Cloud Build, configure a variável de substituição `_WEATHER_API_KEY` com sua chave de API.

### 4. Executar o build

Você pode executar o build manualmente ou deixar que seja acionado automaticamente por commits no repositório, dependendo da configuração da sua trigger.

## Verificação do deploy

Após o deploy, você receberá um URL para o serviço no formato:
```
https://cep-weather-api-[hash].run.app
```

Teste a aplicação com:

```
curl "https://cep-weather-api-[hash].run.app/?cep=01001000"
```

## Monitoramento e logs

Para visualizar os logs da aplicação:

```bash
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=cep-weather-api"
``` 