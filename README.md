# bara

Generate images with **GPT Image 2** from the terminal — billed to **your own**
OpenAI API key — and get every result back as a link.

A self-hosted replacement for the Higgsfield CLI: same shape (`model list`,
`generate create … --wait`, single Go binary, GitHub Releases, npm wrapper), but
every request goes straight to OpenAI with a key you hold. No middleman account,
no platform credits.

## Contents

- [Install](#install)
- [Quickstart](#quickstart)
- [Configuration](#configuration)
- [Examples](#examples)
- [Links (catbox)](#links-catbox)
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
# 1. Store your key (written to credentials.json with 0600 permissions)
bara settings set openai_api_key --stdin      # prompts, keeps it out of shell history

# 2. Generate
bara generate create gpt_image_2 --prompt "a quiet beach at sunrise" --wait
```

```
saved /Users/you/bara-output/openai-1753...-1.png
url   https://files.catbox.moe/ab12cd.png
```

The file is written to `~/bara-output` and published to catbox.moe.

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
| `BARA_OUTPUT_DIR` | where images are saved |
| `BARA_OPENAI_BASE_URL` | point at a proxy or gateway |
| `BARA_CONFIG_DIR` | relocate the whole config directory |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | S3 upload credentials |
| `NO_COLOR` | disable ANSI styling |

## Examples

```bash
bara generate create gpt_image_2 \
  --prompt "modern architecture, glass facade, golden hour" \
  --size 1536x1024 --quality high \
  --wait
```

`gpt-image-2` accepts any size whose edges are multiples of 16, up to 3840 per
edge, within a 3:1 ratio. Run `bara model get gpt_image_2` for every parameter,
or see [MODELS.md](./MODELS.md).

### Editing an image

Passing `--image` switches to the edits endpoint automatically. It takes a local
path or a URL — including a catbox link from an earlier run:

```bash
bara generate create gpt_image_2 \
  --prompt "make the jacket red" \
  --image ./photo.png \
  --wait
```

### Scripting

`--quiet` prints one line per image, the link first when there is one:

```bash
url=$(bara generate create gpt_image_2 --prompt "a red bicycle" --wait --quiet)
bara generate list --json | jq -r '.[] | .outputs[0].public_url // .outputs[0].local_path'
```

## Links (catbox)

OpenAI returns images as base64 with no URL, so bara republishes each result to
**catbox.moe** by default: no account, permanent, up to 200 MB.

catbox links are **public** — anyone who has the URL can open the image — and
anonymous uploads cannot be deleted. For anything private, opt out or use a
bucket you control:

```bash
bara generate create gpt_image_2 --prompt "..." --wait --no-upload   # this run only
bara settings set upload_backend none                                  # always local
bara settings set catbox_userhash --stdin   # tie uploads to your catbox account so you can delete them
```

If catbox is down (it pauses uploads from time to time) the image is still saved
locally and the error is printed; the job does not fail.

| Backend | Account | Retention | Notes |
|---|---|---|---|
| `catbox` | none | permanent | **default**; public |
| `litterbox` | none | 1h–72h | public; good for a quick share |
| `s3` | yours | you decide | Cloudflare R2, AWS S3, Backblaze B2, MinIO |
| `none` | — | — | outputs stay local |

Run `bara upload backends` to see which is active, or force one per run with
`--upload litterbox`.

**Cloudflare R2** is the private-by-default option: 10 GB free and no egress
fees. Create a bucket, enable its public `r2.dev` URL, make an S3 API token,
then:

```bash
bara settings set upload_backend s3
bara settings set s3_endpoint https://<account-id>.r2.cloudflarestorage.com
bara settings set s3_bucket my-renders
bara settings set s3_region auto
bara settings set s3_access_key_id --stdin
bara settings set s3_secret_access_key --stdin
bara settings set upload_base_url https://pub-<hash>.r2.dev
```

Upload files on their own with `bara upload create ./a.png ./b.png`.

## MCP server

The same generation is exposed over the Model Context Protocol, so an agent can
make images as a tool call.

```bash
claude mcp add --scope user bara -- bara mcp serve   # Claude Code, every project
bara mcp config                                      # snippet for other clients
```

Tools: `list_models`, `get_model`, `generate`, `get_job`, `list_jobs`, `upload_file`.

`generate` returns each image's `local_path` and, when publishing succeeds, its
`public_url`. Pass `upload: false` to keep a result local. A failed upload comes
back as `upload_error` next to the local path instead of failing the call. The
raw OpenAI response (which holds the image as base64) is never returned, so a
generation costs a few hundred tokens of context, not megabytes.

Transport is newline-delimited JSON-RPC 2.0 over stdio. Tool failures come back
in-band (`isError: true`) so the model can correct itself rather than seeing a
transport error.

## Commands

| Command | Purpose |
|---|---|
| `bara settings` | list / get / set / unset API keys and preferences |
| `bara model` | list models, inspect a parameter schema |
| `bara generate` | create / cost / get / list / wait jobs |
| `bara upload` | publish files to a public URL, list backends |
| `bara mcp` | run the MCP server, print client config |
| `bara version` | build info |

Global flags: `--json`, `--no-color`.

`generate create` flags: `--wait`, `--wait-timeout` (default `10m`),
`--wait-interval` (default `3s`), `--out`, `--quiet`, `--no-save`, `--json`,
`--upload [backend]`, `--no-upload`.

## How it works

```
bara generate create gpt_image_2 --param value
        |
        +-- catalog        validate params against the model schema (types, enums, ranges)
        +-- provider       POST /images/generations, or /images/edits (multipart) when --image is given
        +-- jobs           record the job locally so get/list work on an id
        +-- output         decode the base64 image into the output dir
        +-- uploader       publish to a URL (catbox by default; litterbox, s3, or none)
```

Parameters are validated **before** the request goes out, so a typo costs
nothing. Unknown flags list what the model actually accepts.

## Development

```bash
go test ./...          # unit tests, no network required
go vet ./...
go build ./cmd/bara
```

The model registry is data — see `internal/catalog/models.json`, validated by
tests. Regenerate the docs with `go run ./tools/genmodels > MODELS.md`.

Releases are cut by GoReleaser on a `v*` tag:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

That builds all six OS/arch combinations, publishes the GitHub Release with
`checksums.txt`, and publishes the npm wrapper when `NPM_TOKEN` is set.

## License

MIT
