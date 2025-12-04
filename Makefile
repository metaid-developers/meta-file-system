.PHONY: build clean run-indexer run-uploader test deps init-db swagger swagger-indexer swagger-uploader docker-build docker-up docker-down docker-logs

# Build all services
build:
	@echo "Building services..."
	@mkdir -p bin
	@go build -o bin/indexer ./cmd/indexer
	@go build -o bin/uploader ./cmd/uploader
	@echo "Build completed!"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin
	@rm -rf data
	@echo "Clean completed!"

# Run indexer service
run-indexer:
	@go run cmd/indexer/main.go --config=conf/conf_loc.yaml

# Run uploader service
run-uploader:
	@go run cmd/uploader/main.go --config=conf/conf_loc.yaml

# Run tests
test:
	@go test -v ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download
	@echo "Dependencies installed!"

# Install tool
install-swag:
	@echo "Installing swag..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Swag installed! Make sure $$GOPATH/bin is in your PATH"

# Generate all Swagger documentation
swagger: swagger-indexer swagger-uploader
	@echo "All Swagger docs generated!"

# Generate Indexer Swagger documentation
swagger-indexer:
	@echo "Generating Indexer Swagger docs..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/indexer/main.go -o docs/indexer --parseDependency --parseInternal --instanceName indexer --tags "Indexer File Query,Indexer PIN Query,Indexer Status,Indexer User Info,Indexer Admin"; \
	elif [ -f ~/go/bin/swag ]; then \
		~/go/bin/swag init -g cmd/indexer/main.go -o docs/indexer --parseDependency --parseInternal --instanceName indexer --tags "Indexer File Query,Indexer PIN Query,Indexer Status,Indexer User Info,Indexer Admin"; \
	elif [ -f $${GOPATH}/bin/swag ]; then \
		$${GOPATH}/bin/swag init -g cmd/indexer/main.go -o docs/indexer --parseDependency --parseInternal --instanceName indexer --tags "Indexer File Query,Indexer PIN Query,Indexer Status,Indexer User Info,Indexer Admin"; \
	else \
		echo "Error: swag not found. Please run 'make install-swag' first"; \
		exit 1; \
	fi
	@echo "Indexer Swagger docs generated at docs/indexer/"

# Generate Uploader Swagger documentation
swagger-uploader:
	@echo "Generating Uploader Swagger docs..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/uploader/main.go -o docs/uploader --parseDependency --parseInternal --instanceName uploader --tags "File Upload,Configuration"; \
	elif [ -f ~/go/bin/swag ]; then \
		~/go/bin/swag init -g cmd/uploader/main.go -o docs/uploader --parseDependency --parseInternal --instanceName uploader --tags "File Upload,Configuration"; \
	elif [ -f $${GOPATH}/bin/swag ]; then \
		$${GOPATH}/bin/swag init -g cmd/uploader/main.go -o docs/uploader --parseDependency --parseInternal --instanceName uploader --tags "File Upload,Configuration"; \
	else \
		echo "Error: swag not found. Please run 'make install-swag' first"; \
		exit 1; \
	fi
	@echo "Uploader Swagger docs generated at docs/uploader/"

# Initialize database
init-db:
	@echo "Initializing database..."
	@mysql -u root -p < scripts/init.sql
	@echo "Database initialized!"

# Docker related commands

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	@cd deploy && docker-compose build
	@echo "Docker images built!"

# Start Docker services (full mode))
docker-up:
	@echo "Starting services with Docker..."
	@cd deploy && docker-compose up -d
	@echo "Services started!"
	@echo "Uploader: http://localhost:7282"
	@echo "Indexer:  http://localhost:7281"

# Start Uploader service
docker-up-uploader:
	@echo "Starting Uploader service..."
	@cd deploy && docker-compose -f docker-compose.uploader.yml up -d
	@echo "Uploader started: http://localhost:7282"

# Start Indexer service
docker-up-indexer:
	@echo "Starting Indexer service..."
	@cd deploy && docker-compose -f docker-compose.indexer.yml up -d
	@echo "Indexer started: http://localhost:7281"

# Stop Docker services
docker-down:
	@echo "Stopping services..."
	@cd deploy && docker-compose down
	@echo "Services stopped!"

# View Docker logs
docker-logs:
	@cd deploy && docker-compose logs -f --tail=100

# View service status
docker-ps:
	@cd deploy && docker-compose ps

# Restart Docker services
docker-restart:
	@echo "Restarting services..."
	@cd deploy && docker-compose restart
	@echo "Services restarted!"

# Complete cleanup (including data volumes))
docker-clean:
	@echo "Cleaning Docker resources..."
	@cd deploy && docker-compose down -v
	@echo "Docker resources cleaned!"

