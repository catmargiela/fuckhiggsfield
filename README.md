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

Media flags take a local path **or** a URL. Local files are inlined as data URIs,
so no bucket is needed.

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

Tools: `list_models`, `get_model`, `generate`, `get_job`, `list_jobs`.

Transport is newline-delimited JSON-RPC 2.0 over stdio. Tool failures come back
in-band (`isError: true`) so the model can correct itself rather than seeing a
transport error.

## Commands

| Command | Purpose |
|---|---|
| `bara settings` | list / get / set / unset API keys and preferences |
| `bara model` | list models, inspect a parameter schema |
| `bara generate` | create / cost / get / list / wait / cancel jobs |
| `bara mcp` | run the MCP server, print client config |
| `bara version` | build info |

Global flags: `--json`, `--no-color`.

`generate create` flags: `--wait`, `--wait-timeout` (default `10m`),
`--wait-interval` (default `3s`), `--out`, `--quiet`, `--no-save`, `--json`.

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
