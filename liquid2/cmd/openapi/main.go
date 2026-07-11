package main

import (
	"fmt"
	"os"

	"github.com/c86j224s/liquid2/internal/app"
	httptransport "github.com/c86j224s/liquid2/internal/transport/http"
)

func main() {
	spec, err := httptransport.OpenAPISpec(app.NewService()).Downgrade()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate OpenAPI spec: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(spec); err != nil {
		fmt.Fprintf(os.Stderr, "write OpenAPI spec: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
}
