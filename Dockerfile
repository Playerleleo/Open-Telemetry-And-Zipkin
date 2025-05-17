FROM golang:1.23 AS builder

WORKDIR /build

# Copiar apenas os arquivos necessários para gerenciar dependências
COPY go.mod .
# COPY go.sum .

# Download de dependências (se houver go.sum descomente a linha abaixo)
# RUN go mod download

# Copiar o código fonte
COPY . .

# Compilar a aplicação com flags de otimização
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main .

# Imagem final - usando alpine mínimo para resolver problemas com certificados
FROM alpine:3.19

# Adicionar certificados CA e apenas o necessário
RUN apk --no-cache add ca-certificates && \
    rm -rf /var/cache/apk/*

# Criar diretório de aplicação
WORKDIR /app

# Copiar apenas o binário compilado
COPY --from=builder /build/main .

# Expor porta
EXPOSE 8080

# Executar a aplicação
CMD ["./main"] 