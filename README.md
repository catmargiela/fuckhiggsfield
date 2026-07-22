# bara

Generate images, video, audio, and text from the terminal — billed to **your own**
OpenAI and RunPod API keys.

A self-hosted replacement for the Higgsfield CLI: same shape (`model list`,
`generate create … --wait`, single Go binary, GitHub Releases, npm wrapper), but
every request goes straight to a provider you hold the key for. No middleman
account, no platform credits.

**41 models** — `gpt-image-2` and the rest of the GPT Image family via OpenAI, and
37 RunPod Public Endpoints covering image, video, audio, and text.

## Contents

- [Install](#install)
- [Quickstart](#quickstart)
- [Configuration](#configuration)
- [Examples](#examples)
- [Models](#models)
- [URL outputs](#getting-a-url-instead-of-a-file)
- [MCP server](#mcp-server)
- [Commands](#commands)
- [How it works](#how-it-works)
- [Development](#development)

## Install

### macOS / Linux — curl

```bash
curl -fsSL https://raw.githubusercontent.com/catmargiela/fuckhiggsfield/main/install.sh | sh
```

The installer verifies the download against the release `checksums.txt`.

### Cross-platform (incl. Windows) — npm

```bash
npm install -g @catmargiela/bara
```

### From source

```bash
go install github.com/catmargiela/fuckhiggsfield/cmd/bara@latest
```

## Quickstart

```bash
# 1. Store your keys (written to credentials.json with 0600 permissions)
bara settings set openai_api_key --stdin      # prompts, keeps it out of shell history
bara settings set runpod_api_key --stdin

# 2. See what you can run
bara model list --image

# 3. Generate
bara generate create gpt_image_2 --prompt "a quiet beach at sunrise" --wait
```

The finished file is written to `~/bara-output` and its path printed.

## Configuration

Settings live in `~/.config/bara/`:

| File | Contents | Mode |
|---|---|---|
| `config.json` | preferences | 0644 |
| `credentials.json` | API keys | 0600 |
| `jobs.json` | local job history | 0600 |

```bash
bara settings list          # secrets are masked
bara settings set output_dir ~/renders
bara settings get openai_api_key --reveal
bara settings path
```

Every setting can be overridden by an environment variable, which takes
precedence over the file and never touches disk:

| Variable | Effect |
|---|---|
| `OPENAI_API_KEY` / `BARA_OPENAI_API_KEY` | OpenAI key |
| `RUNPOD_API_KEY` / `BARA_RUNPOD_API_KEY` | RunPod key |
| `BARA_OUTPUT_DIR` | where media is saved |
| `BARA_OPENAI_BASE_URL` / `BARA_RUNPOD_BASE_URL` | point at a proxy or gateway |
| `BARA_CONFIG_DIR` | relocate the whole config directory |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | S3 upload credentials |
| `NO_COLOR` | disable ANSI styling |

## Examples

### GPT Image 2 (OpenAI)

```bash
bara generate create gpt_image_2 \
  --prompt "modern architecture, glass facade, golden hour" \
  --size 1536x1024 --quality high \
  --wait
```

`gpt-image-2` accepts any size whose edges are multiples of 16, up to 3840 per
edge, within a 3:1 ratio.

### Editing an image

Passing `--image` switches to the edits endpoint automatically:

```bash
bara generate create gpt_image_2 \
  --prompt "make the jacket red" \
  --image ./photo.png \
  --wait
```

### Text to image (RunPod)

```bash
bara generate create flux_dev \
  --prompt "drone shot over a mountain valley at sunrise" \
  --width 1280 --height 720 --num_inference_steps 28 \
  --wait
```

### Image to video

```bash
bara generate create wan_2_6_i2v \
  --prompt "slow camera push through the trees" \
  --image ./still.png \
  --duration 5 --size 1280*720 \
  --wait
```

Media flags take a local path **or** a URL. Local files are inlined as data URIs
by default, so no bucket is needed — see [URL outputs](#getting-a-url-instead-of-a-file)
if you would rather they were uploaded.

### Multi-image edit

```bash
bara generate create nano_banana_pro_edit \
  --prompt "put the product on a marble counter" \
  --images ./product.png --images ./scene.png \
  --resolution 2k --aspect_ratio 4:5 \
  --wait
```

### Text to speech

```bash
bara generate create chatterbox_turbo --prompt "Hello from bara" --voice lucy --wait
```

### Getting a URL instead of a file

OpenAI returns images as base64 with no URL, and RunPod's own result links
expire after 7 days — so by default you get a local file. `--upload` republishes
each result to a public URL you can share:

```bash
bara generate create gpt_image_2 --prompt "a red bicycle" --wait --upload litterbox
```

```
saved /Users/you/bara-output/openai-1753...-1.png
url   https://litter.catbox.moe/ab12cd.png
```

Uploading is **off by default** — nothing leaves your machine until you choose a
backend. Make it automatic:

```bash
bara settings set upload_backend s3
bara generate create flux_dev --prompt "..." --wait     # every result now gets a URL
bara generate create flux_dev --prompt "..." --no-upload  # opt out for one run
```

| Backend | Account | Retention | Notes |
|---|---|---|---|
| `s3` | yours | you decide | Cloudflare R2, AWS S3, Backblaze B2, MinIO — **the durable option** |
| `litterbox` | none | 1h–72h | zero setup, good for a quick share |
| `catbox` | none | permanent | public; the service pauses uploads periodically |

Run `bara upload backends` to see which is active.

**Cloudflare R2** is the recommended target: 10 GB free indefinitely and no
egress fees. Create a bucket, enable its public `r2.dev` URL, make an S3 API
token, then:

```bash
bara settings set upload_backend s3
bara settings set s3_endpoint https://<account-id>.r2.cloudflarestorage.com
bara settings set s3_bucket my-renders
bara settings set s3_region auto
bara settings set s3_access_key_id --stdin
bara settings set s3_secret_access_key --stdin
bara settings set upload_base_url https://pub-<hash>.r2.dev
```

`upload_base_url` is what gets prepended to the object key to build the returned
link, so point it at whatever domain actually serves the bucket. Objects are
keyed `<prefix>/YYYY/MM/DD/<name>` so the bucket stays browsable.

Upload files on their own with `bara upload create ./a.png ./b.png`.

Configuring a backend also changes how **local media inputs** are sent: instead
of being inlined as a data URI, they are uploaded and passed by URL, which
removes the 20 MB inline ceiling.

### Async jobs

`--wait` is optional. Without it you get a job id back:

```bash
id=$(bara generate create seedance_1_5_pro_i2v --prompt "..." --image ./a.png --json | jq -r .id)
bara generate wait "$id"
bara generate list --status COMPLETED
```

### Cost before you spend

```bash
bara generate cost flux_dev --width 1024 --height 1024
bara generate cost vidu_q3_t2v --duration 10
```

This reads the provider's published rate — it is not a quote. Models with tiered
pricing print the rate without inventing a single number.

### Scripting

```bash
bara generate list --json | jq -r '.[] | select(.status=="COMPLETED") | .outputs[0].local_path'
```

## Models

```bash
bara model list                      # everything
bara model list --video              # by kind
bara model list --provider runpod
bara model list --query wan
bara model get wan_2_6_i2v           # full parameter schema
```

Full catalog with every parameter: [MODELS.md](./MODELS.md).

| Kind | Count | Highlights |
|---|---|---|
| Image | 21 | `gpt_image_2`, `flux_dev`, `qwen_image`, `seedream_4_t2i`, `nano_banana_pro_edit`, `z_image_turbo` |
| Video | 16 | `sora_2_i2v`, `wan_2_6_t2v`, `kling_v2_1_i2v_pro`, `seedance_1_5_pro_i2v`, `vidu_q3_t2v` |
| Audio | 2 | `chatterbox_turbo`, `minimax_speech_02_hd` |
| Text | 2 | `qwen3_32b`, `granite_4` |

## MCP server

The same catalog is exposed over the Model Context Protocol, so an agent can
generate media as a tool call.

```bash
bara mcp config                      # prints a client config snippet
claude mcp add bara -- bara mcp serve
```

Tools: `list_models`, `get_model`, `generate`, `get_job`, `list_jobs`, `upload_file`.

Pass `upload: true` to `generate` to get a public URL back instead of only a path.

Transport is newline-delimited JSON-RPC 2.0 over stdio. Tool failures come back
in-band (`isError: true`) so the model can correct itself rather than seeing a
transport error.

## Commands

| Command | Purpose |
|---|---|
| `bara settings` | list / get / set / unset API keys and preferences |
| `bara model` | list models, inspect a parameter schema |
| `bara generate` | create / cost / get / list / wait / cancel jobs |
| `bara upload` | publish files to a public URL, list backends |
| `bara mcp` | run the MCP server, print client config |
| `bara version` | build info |

Global flags: `--json`, `--no-color`.

`generate create` flags: `--wait`, `--wait-timeout` (default `10m`),
`--wait-interval` (default `3s`), `--out`, `--quiet`, `--no-save`, `--json`,
`--upload [backend]`, `--no-upload`.

## How it works

```
bara generate create <model> --param value
        |
        +-- catalog        validate params against the model schema (types, enums, ranges)
        +-- media          local paths -> data URIs (RunPod) or multipart (OpenAI)
        +-- provider
        |     openai       POST /images/generations, or /images/edits when --image is given
        |     runpod       POST /v2/<endpoint>/run, poll /v2/<endpoint>/status/<id>
        +-- jobs           record the job locally so get/list/wait work on an id
        +-- output         decode base64 or download the URL into the output dir
        +-- uploader       optional: republish to a durable URL (s3/R2, litterbox, catbox)
```

Parameters are validated **before** the request goes out, so a typo costs
nothing. Unknown flags list what the model actually accepts.

RunPod output shapes differ per model, so results are extracted structurally
(any `*_url` key, `images` arrays, base64 blobs, chat-style `choices`) rather
than against one fixed schema.

## Development

```bash
go test ./...          # unit tests, no network required
go vet ./...
go build ./cmd/bara
```

Adding a model is a JSON edit — see `internal/catalog/models.json`. The registry
is validated by tests: every entry needs a provider, kind, endpoint slug, and a
parameter list whose defaults satisfy their own enums.

Releases are cut by GoReleaser on a `v*` tag:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

That builds all six OS/arch combinations, publishes the GitHub Release with
`checksums.txt`, and publishes the npm wrapper when `NPM_TOKEN` is set.

## License

MIT
