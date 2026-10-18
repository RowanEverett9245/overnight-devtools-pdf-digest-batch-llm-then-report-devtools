package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/example/overnight-devtools-digest/digest"
)

func main() {
	batchRequestPath := flag.String("batch-request", "", "JSON body for a new batch submission")
	batchID := flag.String("batch-id", "", "completed batch id to render")
	flag.Parse()

	client, err := digest.NewClient(os.Getenv("INFRAI_API_KEY"))
	if err != nil {
		fatal(err)
	}
	ctx := context.Background()
	if *batchRequestPath != "" {
		body, err := os.ReadFile(*batchRequestPath)
		if err != nil {
			fatal(err)
		}
		if !json.Valid(body) {
			fatal(fmt.Errorf("%s is not JSON", *batchRequestPath))
		}
		id, err := client.SubmitBatch(ctx, body)
		if err != nil {
			fatal(err)
		}
		fmt.Println(id)
		return
	}
	if *batchID == "" {
		fatal(fmt.Errorf("provide -batch-request or -batch-id"))
	}
	results, err := client.BatchResults(ctx, *batchID)
	if err != nil {
		fatal(err)
	}
	pdf, err := client.RenderDigest(ctx, digest.MarkdownFromBatchResults(results))
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(pdf))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
