
run::
	(cd backend && go generate ./cmd/server) && (cd app && npm run api:generate)

setup::
	cp .env.example .env
	cp app/.env.example app/.env
