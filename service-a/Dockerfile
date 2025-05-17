FROM golang:1.24 AS builder

WORKDIR /build

# Copiar os arquivos de módulo
COPY go.mod .

# Copiar o código fonte
COPY . .

# Baixar as dependências
RUN go mod download

# Compilar a aplicação
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main ./cmd

# Imagem final
FROM alpine:3.19

# Adicionar certificados CA
RUN apk --no-cache add ca-certificates && \
    rm -rf /var/cache/apk/*

# Criar diretório de aplicação
WORKDIR /app

# Copiar apenas o binário compilado
COPY --from=builder /build/main .

# Expor porta
EXPOSE 8081

# Executar a aplicação
CMD ["./main"] 