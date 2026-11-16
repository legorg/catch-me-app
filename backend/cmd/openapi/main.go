// Command openapi writes the OpenAPI document of the API.
package main

import (
	"flag"
	"log"
	"os"

	"catch-me/backend/api"
)

func main() {
	out := flag.String("out", "openapi.json", "file to write the OpenAPI document to")
	flag.Parse()
	spec, err := api.OpenAPI()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, append(spec, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
}
