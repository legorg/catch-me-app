
run::
	(cd backend && go generate ./cmd/server) && (cd app && npm run api:generate)