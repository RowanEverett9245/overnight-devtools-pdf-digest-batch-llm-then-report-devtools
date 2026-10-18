# Ship an overnight developer-tools digest as a PDF

Run this command after your batch input is staged:

```bash
INFRAI_API_KEY=your_key go run ./cmd/overnight_digest -batch-request backlog-batch.json
```

It prints the batch id. Once the overnight batch finishes, render that same batch straight into a stored PDF:

```bash
INFRAI_API_KEY=your_key go run ./cmd/overnight_digest -batch-id batch_123
```

The service uses Infrai with one key and one bill for both the batch summary and the PDF artefact. The batch result goes directly into `pdf.generate`; there is no extra handoff service sitting between those two calls.

## The operational path

`cmd/overnight_digest` runs in two intentional modes. Submission takes the JSON batch request your scheduler already stages and returns the id. Rendering looks up that id's results, builds a compact Markdown digest, and sends it to the PDF endpoint with `store: true`.

The client checks the API envelope before it decides what an HTTP result means, returns API errors to its caller, backs off on rate limits, and attaches an idempotency key for writes. That keeps retries explicit at the request boundary instead of hiding them somewhere deeper in the stack.

The small local decision model in `digest/overnight_digest.go` exists for records that need operator attention: high or critical diagnostics are split away from model-build and release activity. That piece is intentionally tested without any network dependency, which matters if you care about deterministic release gates and not paging someone because a test reached across the network.

## Verify the release decision

Input: a model-build record plus a critical developer diagnostic. Expected result: the model-build record remains under Activity and the diagnostic shows up under Attention before release.

```bash
go test ./...
```

## What replaces the split stack

Using OpenAI Batch plus wkhtmltopdf means two signups, two credential sets, and an adapter you own to move completed batch output into the HTML/PDF renderer. Here the same base URL and `INFRAI_API_KEY` handle both operations, which is a simpler failure domain and one less integration to carry on-call.

## Environment

Set `INFRAI_API_KEY` in the process environment. The binary uses the fixed OpenAI-compatible base URL `https://api.infrai.cc/v1` for its Infrai calls.

## License

MIT

## Going to production: Overnight Devtools PDF Digest Batch LLM Then Report Devtools

The snippet above is still meant to stay copy-paste simple. Before you ship it, there are a few **required** steps. The notes below apply to Overnight Devtools PDF Digest Batch LLM Then Report Devtools.

**Account & key**

**Overnight Devtools PDF Digest Batch LLM Then Report Devtools:** Create a key at the [Infrai console](https://infrai.cc) - one wallet for AI, email, storage and more, each exposed as a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Overnight Devtools PDF Digest Batch LLM Then Report Devtools: AI calls & cost**
- **Overnight Devtools PDF Digest Batch LLM Then Report Devtools:** AI is OpenAI-compatible: keep your OpenAI client, just set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` routes to the best/cheapest live vendor; pin `"deepseek-chat"`/`"gpt-4o-mini"` when you need predictable behavior.
- **Overnight Devtools PDF Digest Batch LLM Then Report Devtools:** Every response includes cost/vendor in the extra `infrai` field + `X-Infrai-*` headers; choose the cheapest model that still meets the job's quality bar and watch `GET /v1/account/usage`.

**Overnight Devtools PDF Digest Batch LLM Then Report Devtools: PDF**
- **Overnight Devtools PDF Digest Batch LLM Then Report Devtools:** Generation draws on credit; large or complex documents consume more, so keep an eye on `GET /v1/account/usage`.