# Variáveis de configuração
DB_URL=postgres://myuser:mysecretpassword@localhost:5432/mydb?sslmode=disable

migrate-up:
	migrate -path db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DB_URL)" down
