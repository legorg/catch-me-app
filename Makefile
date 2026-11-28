
run::
	(cd backend && go run ./cmd/server)
generate::
	(cd backend && go generate ./cmd/server) && (cd app && npm run api:generate)

android::
	nix develop --command npm --prefix app run android

ios::
	nix develop --command npm --prefix app run ios

# Release builds against the production API, with the JavaScript bundled in so they run without Metro.
# An exported EXPO_PUBLIC_API_URL wins over .env; override with make android-prod PROD_API_URL=https://….
PROD_API_URL ?= https://xyz.l3montree.cloud

android-prod::
	EXPO_PUBLIC_API_URL=$(PROD_API_URL) nix develop --command npm --prefix app run android:prod

ios-prod::
	EXPO_PUBLIC_API_URL=$(PROD_API_URL) nix develop --command npm --prefix app run ios:prod

clean::
	nix develop --command npm --prefix app run clean

setup::
	cp .env.example .env

# Advisory dead-code report; not wired into anything else.
deadcode::
	cd backend && go run honnef.co/go/tools/cmd/staticcheck@2026.1 -checks U1000 ./...
	cd backend && go run honnef.co/go/tools/cmd/staticcheck@2026.1 -checks U1000 -tests=false ./...
	cd backend && go run golang.org/x/tools/cmd/deadcode@v0.38.0 ./cmd/server ./cmd/openapi
	cd backend && go mod tidy -diff
