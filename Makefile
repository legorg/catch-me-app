
run::
	(cd backend && go generate ./cmd/server) && (cd app && npm run api:generate)

setup::
	cp .env.example .env
	cp app/.env.example app/.env

# Advisory dead-code report; not wired into anything else.
deadcode::
	cd backend && go run honnef.co/go/tools/cmd/staticcheck@2026.1 -checks U1000 ./...
	cd backend && go run honnef.co/go/tools/cmd/staticcheck@2026.1 -checks U1000 -tests=false ./...
	cd backend && go run golang.org/x/tools/cmd/deadcode@v0.38.0 ./cmd/server ./cmd/openapi
	cd backend && go mod tidy -diff
