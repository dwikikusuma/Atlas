# Makefile

# ==============================================================================
# VARIABLES
# ==============================================================================
PROTOC_IMAGE = rvolosatovs/protoc:4.0.0
# Updated to match the atlas postgres container credentials
DB_URL=postgres://atlas:atlaspassword@localhost:5432/atlas_db?sslmode=disable

# Default to 'tracker' service if not specified (since it's the high-traffic one)
SERVICE ?= tracker
MIGRATION_PATH=internal/$(SERVICE)/db/migration

# ==============================================================================
# COMMANDS
# ==============================================================================

.PHONY: all proto clean up down help

help: ## Show this help message
	@echo 'Usage:'
	@echo '  make proto   - Generate Go code from .proto files using Docker'
	@echo '  make up      - Start the infrastructure (Docker Compose)'
	@echo '  make down    - Stop the infrastructure'
	@echo '  make clean   - Remove generated code'
	@echo '  make sqlc    - Generate DB code using SQLC'

# 2. Infrastructure
up:
	@echo "🐳 Starting Atlas containers..."
	docker-compose up -d

down:
	@echo "🛑 Stopping Atlas containers..."
	docker-compose down

# 3. Cleanup
clean:
	rm -rf pkg/pb/*

# ==============================================================================
# 1. MIGRATION COMMANDS (Dynamic)
# ==============================================================================
# Usage: make migrate-create name=init_schema service=tracker
migrate-create:
	@echo "📁 Creating migration files for [$(SERVICE)]..."
	@mkdir -p $(MIGRATION_PATH)
	docker run --rm -v $(CURDIR)/$(MIGRATION_PATH):/migrations --network host migrate/migrate \
		create -ext sql -dir /migrations -seq $(name)

# Usage: make migrate-up service=tracker
migrate-up:
	@echo "🚀 Running Migrations Up for [$(SERVICE)]..."
	docker run --rm -v $(CURDIR)/$(MIGRATION_PATH):/migrations --network host migrate/migrate \
		-path=/migrations/ -database "$(DB_URL)" up

# Usage: make migrate-down service=tracker
migrate-down:
	@echo "🔙 Running Migrations Down for [$(SERVICE)]..."
	docker run --rm -v $(CURDIR)/$(MIGRATION_PATH):/migrations --network host migrate/migrate \
		-path=/migrations/ -database "$(DB_URL)" down 1

# ==============================================================================
# 2. CODE GENERATION (SQLC & Proto)
# ==============================================================================

# Generates SQLC for ALL services defined in sqlc.yaml
# We use the Docker container to ensure everyone uses the same version
sqlc:
	@echo "🤖 Generating SQLC code..."
	docker run --rm -v $(CURDIR):/src -w /src kjconroy/sqlc generate
	@echo "✅ SQLC Generation Complete!"

# Generates Protobufs
# Updated to include the new Atlas services
proto:
	@echo "🚀 Generating gRPC code..."
	@mkdir -p pkg/pb
	docker run --rm -v $(CURDIR):/workspace -w /workspace $(PROTOC_IMAGE) \
		--proto_path=api/proto \
		--go_out=pkg/pb --go_opt=paths=source_relative \
		--go-grpc_out=pkg/pb --go-grpc_opt=paths=source_relative \
		tracker/tracker.proto \
		dispatch/dispatch.proto
		@echo "✅ Proto Generation Complete!"
#		order/order.proto \
#		wallet/wallet.proto \