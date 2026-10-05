# Models

Generated from `internal/catalog/models.json` by `go run ./tools/genmodels`.
Do not edit by hand.

Run `bara model get <model>` for the same schema in your terminal.

## Index

### Image (1)

| model | provider | name | price |
|---|---|---|---|
| [`gpt_image_2`](#gptimage2) | openai | GPT Image 2 | billed per image by size/quality (see OpenAI pricing) |

## Reference

### gpt_image_2

**GPT Image 2** — image via openai

- Endpoint: `gpt-image-2`
- Price: billed per image by size/quality (see OpenAI pricing)

Flexible sizes: each edge a multiple of 16, max edge 3840, ratio within 3:1, total pixels 655360-8294400. Returns base64; input_fidelity is always high and cannot be set. Pass --image to edit instead of generate.

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  | text description (up to 32000 chars) |
| `--image` | string[] |  |  | image path or URL | input image(s) — switches to the edits endpoint |
| `--mask` | string |  |  | image path or URL | PNG mask; applied to the first input image |
| `--n` | int |  | 1 | 1–10 | number of images |
| `--size` | string |  | auto |  | auto, 1024x1024, 1536x1024, 1024x1536, or WIDTHxHEIGHT |
| `--quality` | string |  | auto | `auto`, `low`, `medium`, `high` | render quality |
| `--background` | string |  |  | `transparent`, `opaque`, `auto` | background handling |
| `--output_format` | string |  | png | `png`, `jpeg`, `webp` | encoding of the returned image |
| `--output_compression` | int |  |  | 0–100 | compression percent for jpeg/webp |
| `--moderation` | string |  |  | `auto`, `low` | moderation strictness |
| `--user` | string |  |  |  | end-user identifier for abuse tracking |

```bash
bara generate create gpt_image_2 --prompt "..." --wait
```

