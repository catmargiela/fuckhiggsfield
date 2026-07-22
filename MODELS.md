# Models

Generated from `internal/catalog/models.json` by `go run ./tools/genmodels`.
Do not edit by hand.

Run `bara model get <model>` for the same schema in your terminal.

## Index

### Image (21)

| model | provider | name | price |
|---|---|---|---|
| [`flux_dev`](#fluxdev) | runpod | FLUX.1 dev | $0.02 per megapixel |
| [`flux_kontext_dev`](#fluxkontextdev) | runpod | FLUX.1 Kontext dev | $0.025 per image |
| [`flux_schnell`](#fluxschnell) | runpod | FLUX.1 schnell | $0.0024 per megapixel |
| [`gpt_image_1`](#gptimage1) | openai | GPT Image 1 | billed per image by size/quality (see OpenAI pricing) |
| [`gpt_image_1_5`](#gptimage15) | openai | GPT Image 1.5 | billed per image by size/quality (see OpenAI pricing) |
| [`gpt_image_1_mini`](#gptimage1mini) | openai | GPT Image 1 Mini | cheapest GPT Image tier (see OpenAI pricing) |
| [`gpt_image_2`](#gptimage2) | openai | GPT Image 2 | billed per image by size/quality (see OpenAI pricing) |
| [`nano_banana_2_edit`](#nanobanana2edit) | runpod | Nano Banana 2 Edit | $0.0875 (1k) / $0.13 (2k) / $0.175 (4k) per image |
| [`nano_banana_edit`](#nanobananaedit) | runpod | Nano Banana Edit | $0.038 per image edited |
| [`nano_banana_pro_edit`](#nanobananaproedit) | runpod | Nano Banana Pro Edit | $0.14 (1k/2k) - $0.24 (4k) per image |
| [`p_image_edit`](#pimageedit) | runpod | P-Image Edit | $0.01 per image edited |
| [`p_image_t2i`](#pimaget2i) | runpod | P-Image T2I | $0.005 per image |
| [`qwen_image`](#qwenimage) | runpod | Qwen Image | $0.02 per image |
| [`qwen_image_edit`](#qwenimageedit) | runpod | Qwen Image Edit | $0.02 per image edited |
| [`qwen_image_edit_2511`](#qwenimageedit2511) | runpod | Qwen Image Edit 2511 | $0.02 per image edited |
| [`qwen_image_edit_2511_lora`](#qwenimageedit2511lora) | runpod | Qwen Image Edit 2511 LoRA | $0.025 per image edited |
| [`qwen_image_lora`](#qwenimagelora) | runpod | Qwen Image LoRA | $0.025 per image |
| [`seedream_4_edit`](#seedream4edit) | runpod | Seedream 4.0 Edit | $0.027 per image edited |
| [`seedream_4_t2i`](#seedream4t2i) | runpod | Seedream 4.0 T2I | $0.027 per image |
| [`wan_2_6_t2i`](#wan26t2i) | runpod | WAN 2.6 T2I | $0.03 per image |
| [`z_image_turbo`](#zimageturbo) | runpod | Z-Image Turbo | $0.005 per image |

### Video (16)

| model | provider | name | price |
|---|---|---|---|
| [`infinitetalk`](#infinitetalk) | runpod | InfiniteTalk | 480p $0.25 / 720p $0.50 per video |
| [`kling_v2_1_i2v_pro`](#klingv21i2vpro) | runpod | Kling v2.1 I2V Pro | $0.45 per 5s, $0.90 per 10s |
| [`kling_v2_6_motion_control`](#klingv26motioncontrol) | runpod | Kling v2.6 Motion Control | $0.21 (1-3s) / $0.42 (4-6s) / $0.63 (7-9s) |
| [`kling_video_o1_r2v`](#klingvideoo1r2v) | runpod | Kling Video O1 R2V | $0.112 per second |
| [`p_video`](#pvideo) | runpod | Pruna Video | 720p $0.02/s ($0.005 draft), 1080p $0.04/s ($0.01 draft) |
| [`seedance_1_5_pro_i2v`](#seedance15proi2v) | runpod | Seedance 1.5 Pro I2V | $0.024/sec (480p) or $0.052/sec (720p) |
| [`sora_2_i2v`](#sora2i2v) | runpod | SORA 2 I2V | $0.40 (4s) / $0.80 (8s) / $1.20 (12s) |
| [`sora_2_pro_i2v`](#sora2proi2v) | runpod | SORA 2 Pro I2V | $1.20 - $6.00 per video (by resolution and duration) |
| [`vidu_q3_i2v`](#viduq3i2v) | runpod | Vidu Q3 I2V | $0.15 per second |
| [`vidu_q3_t2v`](#viduq3t2v) | runpod | Vidu Q3 T2V | $0.15 per second |
| [`wan_2_1_i2v`](#wan21i2v) | runpod | WAN 2.1 I2V 720p | $0.30 per 5 seconds |
| [`wan_2_2_i2v`](#wan22i2v) | runpod | WAN 2.2 I2V 720p | $0.30 - $0.90 per video (5-15s) |
| [`wan_2_2_i2v_lora`](#wan22i2vlora) | runpod | WAN 2.2 I2V 720p LoRA | $0.35 (5s) / $0.56 (8s) |
| [`wan_2_5`](#wan25) | runpod | WAN 2.5 I2V | $0.25 - $1.50 per video (480p/5s to 1080p/10s) |
| [`wan_2_6_i2v`](#wan26i2v) | runpod | WAN 2.6 I2V | 720p $0.10/sec, 1080p $0.15/sec |
| [`wan_2_6_t2v`](#wan26t2v) | runpod | WAN 2.6 T2V | 720p $0.50-$1.50 / 1080p $0.75-$2.25 by duration |

### Audio (2)

| model | provider | name | price |
|---|---|---|---|
| [`chatterbox_turbo`](#chatterboxturbo) | runpod | Chatterbox Turbo | $0.001 per second of audio |
| [`minimax_speech_02_hd`](#minimaxspeech02hd) | runpod | Minimax Speech 02 HD | $0.05 per 1000 characters |

### Text (2)

| model | provider | name | price |
|---|---|---|---|
| [`granite_4`](#granite4) | runpod | IBM Granite 4.0 H Small | $10.00 per 1M tokens |
| [`qwen3_32b`](#qwen332b) | runpod | Qwen3 32B AWQ | $10.00 per 1M tokens |

## Reference

### flux_dev

**FLUX.1 dev** — image via runpod

- Endpoint: `black-forest-labs-flux-1-dev`
- Price: $0.02 per megapixel

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--negative_prompt` | string |  |  |  |  |
| `--width` | int |  | 1024 | 256–1536 | divisible by 64 |
| `--height` | int |  | 1024 | 256–1536 | divisible by 64 |
| `--num_inference_steps` | int |  | 28 | 1–50 |  |
| `--guidance` | float |  | 7.5 | 0–10 |  |
| `--seed` | int |  | -1 |  | -1 for random |
| `--image_format` | string |  | jpeg | `png`, `jpeg` |  |

```bash
bara generate create flux_dev --prompt "..." --wait
```

### flux_kontext_dev

**FLUX.1 Kontext dev** — image via runpod

- Endpoint: `black-forest-labs-flux-1-kontext-dev`
- Price: $0.025 per image

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--negative_prompt` | string |  |  |  |  |
| `--size` | string |  | 1024*1024 |  | width*height |
| `--num_inference_steps` | int |  | 28 | 1–50 |  |
| `--guidance` | float |  | 2 | 0–10 |  |
| `--seed` | int |  | -1 |  |  |
| `--output_format` | string |  | png | `png`, `jpeg` |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create flux_kontext_dev --prompt "..." --wait
```

### flux_schnell

**FLUX.1 schnell** — image via runpod

- Endpoint: `black-forest-labs-flux-1-schnell`
- Price: $0.0024 per megapixel

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--negative_prompt` | string |  |  |  |  |
| `--width` | int |  | 1024 | 256–1536 |  |
| `--height` | int |  | 1024 | 256–1536 |  |
| `--num_inference_steps` | int |  | 4 | 1–8 |  |
| `--guidance` | float |  | 7.5 | 0–10 |  |
| `--seed` | int |  | -1 |  |  |
| `--image_format` | string |  | jpeg | `png`, `jpeg` |  |

```bash
bara generate create flux_schnell --prompt "..." --wait
```

### gpt_image_1

**GPT Image 1** — image via openai

- Endpoint: `gpt-image-1`
- Price: billed per image by size/quality (see OpenAI pricing)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string[] |  |  | image path or URL |  |
| `--mask` | string |  |  | image path or URL |  |
| `--n` | int |  | 1 | 1–10 |  |
| `--size` | string |  | auto | `auto`, `1024x1024`, `1536x1024`, `1024x1536` |  |
| `--quality` | string |  | auto | `auto`, `low`, `medium`, `high` |  |
| `--background` | string |  |  | `transparent`, `opaque`, `auto` |  |
| `--output_format` | string |  | png | `png`, `jpeg`, `webp` |  |
| `--output_compression` | int |  |  | 0–100 |  |
| `--moderation` | string |  |  | `auto`, `low` |  |
| `--user` | string |  |  |  |  |

```bash
bara generate create gpt_image_1 --prompt "..." --wait
```

### gpt_image_1_5

**GPT Image 1.5** — image via openai

- Endpoint: `gpt-image-1.5`
- Price: billed per image by size/quality (see OpenAI pricing)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  | text description |
| `--image` | string[] |  |  | image path or URL | input image(s) — switches to the edits endpoint |
| `--mask` | string |  |  | image path or URL | PNG mask; applied to the first input image |
| `--n` | int |  | 1 | 1–10 |  |
| `--size` | string |  | auto | `auto`, `1024x1024`, `1536x1024`, `1024x1536` |  |
| `--quality` | string |  | auto | `auto`, `low`, `medium`, `high` |  |
| `--background` | string |  |  | `transparent`, `opaque`, `auto` |  |
| `--output_format` | string |  | png | `png`, `jpeg`, `webp` |  |
| `--output_compression` | int |  |  | 0–100 |  |
| `--input_fidelity` | string |  |  | `low`, `high` | how closely edits preserve the input |
| `--moderation` | string |  |  | `auto`, `low` |  |
| `--user` | string |  |  |  |  |

```bash
bara generate create gpt_image_1_5 --prompt "..." --wait
```

### gpt_image_1_mini

**GPT Image 1 Mini** — image via openai

- Endpoint: `gpt-image-1-mini`
- Price: cheapest GPT Image tier (see OpenAI pricing)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string[] |  |  | image path or URL |  |
| `--mask` | string |  |  | image path or URL |  |
| `--n` | int |  | 1 | 1–10 |  |
| `--size` | string |  | auto | `auto`, `1024x1024`, `1536x1024`, `1024x1536` |  |
| `--quality` | string |  | auto | `auto`, `low`, `medium`, `high` |  |
| `--background` | string |  |  | `transparent`, `opaque`, `auto` |  |
| `--output_format` | string |  | png | `png`, `jpeg`, `webp` |  |
| `--output_compression` | int |  |  | 0–100 |  |
| `--moderation` | string |  |  | `auto`, `low` |  |
| `--user` | string |  |  |  |  |

```bash
bara generate create gpt_image_1_mini --prompt "..." --wait
```

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

### nano_banana_2_edit

**Nano Banana 2 Edit** — image via runpod

- Endpoint: `google-nano-banana-2-edit`
- Price: $0.0875 (1k) / $0.13 (2k) / $0.175 (4k) per image

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--images` | string[] | yes |  | image path or URL | up to 14 image URLs |
| `--resolution` | string |  | 1k | `1k`, `2k`, `4k` |  |
| `--aspect_ratio` | string |  |  | `1:1`, `3:2`, `16:9`, `9:16`, `4:5` |  |
| `--output_format` | string |  | png | `png`, `jpeg` |  |
| `--enable_base64_output` | bool |  | false |  |  |

```bash
bara generate create nano_banana_2_edit --prompt "..." --wait
```

### nano_banana_edit

**Nano Banana Edit** — image via runpod

- Endpoint: `nano-banana-edit`
- Price: $0.038 per image edited

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--images` | string[] | yes |  | image path or URL | up to 4 image URLs |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create nano_banana_edit --prompt "..." --wait
```

### nano_banana_pro_edit

**Nano Banana Pro Edit** — image via runpod

- Endpoint: `nano-banana-pro-edit`
- Price: $0.14 (1k/2k) - $0.24 (4k) per image

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--images` | string[] | yes |  | image path or URL | up to 14 image URLs |
| `--resolution` | string |  | 2k | `1k`, `2k`, `4k` |  |
| `--aspect_ratio` | string |  | 1:1 | `1:1`, `3:2`, `2:3`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9` |  |
| `--output_format` | string |  | png | `png`, `jpeg` |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create nano_banana_pro_edit --prompt "..." --wait
```

### p_image_edit

**P-Image Edit** — image via runpod

- Endpoint: `p-image-edit`
- Price: $0.01 per image edited

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--images` | string[] | yes |  | image path or URL | 1-5 image URLs |
| `--aspect_ratio` | string |  | match_input_image | `match_input_image`, `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3` |  |
| `--seed` | int |  | -1 |  |  |
| `--disable_safety_checker` | bool |  | false |  |  |

```bash
bara generate create p_image_edit --prompt "..." --wait
```

### p_image_t2i

**P-Image T2I** — image via runpod

- Endpoint: `p-image-t2i`
- Price: $0.005 per image

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--aspect_ratio` | string |  | 16:9 | `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3` |  |
| `--seed` | int |  |  |  |  |
| `--disable_safety_checker` | bool |  | false |  |  |

```bash
bara generate create p_image_t2i --prompt "..." --wait
```

### qwen_image

**Qwen Image** — image via runpod

- Endpoint: `qwen-image-t2i`
- Price: $0.02 per image

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--negative_prompt` | string |  |  |  |  |
| `--size` | string |  | 1024*1024 |  | width*height |
| `--seed` | int |  | -1 |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create qwen_image --prompt "..." --wait
```

### qwen_image_edit

**Qwen Image Edit** — image via runpod

- Endpoint: `qwen-image-edit`
- Price: $0.02 per image edited

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--negative_prompt` | string |  |  |  |  |
| `--seed` | int |  | -1 |  |  |
| `--output_format` | string |  | jpeg | `png`, `jpeg` |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create qwen_image_edit --prompt "..." --wait
```

### qwen_image_edit_2511

**Qwen Image Edit 2511** — image via runpod

- Endpoint: `qwen-image-edit-2511`
- Price: $0.02 per image edited

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--images` | string[] | yes |  | image path or URL | 1-3 image URLs |
| `--size` | string |  | 1024*1024 | `1024*1024`, `1024*1280`, `1280*1024`, `1280*1280`, `1280*1536`, `1536*1080` |  |
| `--seed` | int |  | -1 |  |  |
| `--output_format` | string |  | jpeg | `jpeg`, `png`, `webp` |  |

```bash
bara generate create qwen_image_edit_2511 --prompt "..." --wait
```

### qwen_image_edit_2511_lora

**Qwen Image Edit 2511 LoRA** — image via runpod

- Endpoint: `qwen-image-edit-2511-lora`
- Price: $0.025 per image edited

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--images` | string[] | yes |  | image path or URL |  |
| `--loras` | json |  |  |  | [{path, scale}] |
| `--size` | string |  | 1024*1024 | `1024*1024`, `1024*1280`, `1280*1024`, `1280*1280`, `1280*1536`, `1536*1080` |  |
| `--seed` | int |  | -1 |  |  |
| `--output_format` | string |  | jpeg | `jpeg`, `png`, `webp` |  |

```bash
bara generate create qwen_image_edit_2511_lora --prompt "..." --wait
```

### qwen_image_lora

**Qwen Image LoRA** — image via runpod

- Endpoint: `qwen-image-t2i-lora`
- Price: $0.025 per image

--loras takes JSON, e.g. '[{"path":"https://...","scale":0.8}]'

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--loras` | json |  |  |  | [{path, scale}] |
| `--size` | string |  | 1024*1024 |  |  |
| `--seed` | int |  | -1 |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create qwen_image_lora --prompt "..." --wait
```

### seedream_4_edit

**Seedream 4.0 Edit** — image via runpod

- Endpoint: `seedream-v4-edit`
- Price: $0.027 per image edited

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--images` | string[] | yes |  | image path or URL |  |
| `--size` | string |  | 1024*1024 |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create seedream_4_edit --prompt "..." --wait
```

### seedream_4_t2i

**Seedream 4.0 T2I** — image via runpod

- Endpoint: `seedream-v4-t2i`
- Price: $0.027 per image

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--negative_prompt` | string |  |  |  |  |
| `--size` | string |  | 1024*1024 |  | width*height |
| `--seed` | int |  | -1 |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create seedream_4_t2i --prompt "..." --wait
```

### wan_2_6_t2i

**WAN 2.6 T2I** — image via runpod

- Endpoint: `wan-2-6-t2i`
- Price: $0.03 per image

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--size` | string |  | 1024*1024 | `1024*1024`, `1024*768`, `1440*1024` |  |
| `--seed` | int |  | -1 |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create wan_2_6_t2i --prompt "..." --wait
```

### z_image_turbo

**Z-Image Turbo** — image via runpod

- Endpoint: `z-image-turbo`
- Price: $0.005 per image

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  | English or Chinese |
| `--image` | string |  |  | image path or URL | source image for image-to-image |
| `--size` | string |  | 1024*1024 | `512*512`, `768*768`, `1024*1024`, `1280*1280`, `1024*768`, `768*1024`, `1280*720`, `720*1280` |  |
| `--strength` | float |  | 0.8 | 0–1 |  |
| `--seed` | int |  | -1 |  |  |
| `--output_format` | string |  | png | `png`, `jpeg`, `webp` |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create z_image_turbo --prompt "..." --wait
```

### infinitetalk

**InfiniteTalk** — video via runpod

- Endpoint: `infinitetalk`
- Price: 480p $0.25 / 720p $0.50 per video

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--audio` | string | yes |  | audio path or URL |  |
| `--size` | string | yes |  | `480p`, `720p` |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create infinitetalk --prompt "..." --wait
```

### kling_v2_1_i2v_pro

**Kling v2.1 I2V Pro** — video via runpod

- Endpoint: `kling-v2-1-i2v-pro`
- Price: $0.45 per 5s, $0.90 per 10s

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--negative_prompt` | string |  |  |  |  |
| `--guidance_scale` | float |  | 0.5 |  |  |
| `--duration` | int |  | 5 |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create kling_v2_1_i2v_pro --prompt "..." --wait
```

### kling_v2_6_motion_control

**Kling v2.6 Motion Control** — video via runpod

- Endpoint: `kling-v2-6-std-motion-control`
- Price: $0.21 (1-3s) / $0.42 (4-6s) / $0.63 (7-9s)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--image` | string | yes |  | image path or URL |  |
| `--video` | string | yes |  | video path or URL | motion reference video |
| `--character_orientation` | string | yes |  | `video`, `image` |  |
| `--prompt` | string |  |  |  |  |
| `--negative_prompt` | string |  |  |  |  |
| `--keep_original_sound` | bool |  | true |  |  |

```bash
bara generate create kling_v2_6_motion_control --prompt "..." --wait
```

### kling_video_o1_r2v

**Kling Video O1 R2V** — video via runpod

- Endpoint: `kling-video-o1-r2v`
- Price: $0.112 per second

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--images` | string[] | yes |  | image path or URL | 1-10 reference images |
| `--video` | string |  |  | video path or URL |  |
| `--negative_prompt` | string |  |  |  |  |
| `--aspect_ratio` | string |  | 16:9 | `16:9`, `9:16`, `1:1` |  |
| `--duration` | int |  | 5 | 3–10 |  |
| `--seed` | int |  | -1 |  |  |
| `--enable_prompt_expansion` | bool |  | false |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create kling_video_o1_r2v --prompt "..." --wait
```

### p_video

**Pruna Video** — video via runpod

- Endpoint: `p-video`
- Price: 720p $0.02/s ($0.005 draft), 1080p $0.04/s ($0.01 draft)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string |  |  | image path or URL |  |
| `--audio` | string |  |  | audio path or URL |  |
| `--duration` | int |  | 5 | 1–10 |  |
| `--resolution` | string |  | 720p | `720p`, `1080p` |  |
| `--fps` | int |  | 24 | `24`, `48` |  |
| `--aspect_ratio` | string |  | 16:9 | `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `1:1` |  |
| `--seed` | int |  |  |  |  |
| `--draft` | bool |  | false |  |  |
| `--save_audio` | bool |  | true |  |  |
| `--prompt_upsampling` | bool |  | true |  |  |

```bash
bara generate create p_video --prompt "..." --wait
```

### seedance_1_5_pro_i2v

**Seedance 1.5 Pro I2V** — video via runpod

- Endpoint: `seedance-v1-5-pro-i2v`
- Price: $0.024/sec (480p) or $0.052/sec (720p)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--last_image` | string |  |  | image path or URL | optional end frame |
| `--duration` | int |  | 5 | 4–12 |  |
| `--resolution` | string |  | 720p | `480p`, `720p` |  |
| `--aspect_ratio` | string |  | 16:9 | `21:9`, `16:9`, `9:16`, `1:1`, `4:3`, `3:4` |  |
| `--camera_fixed` | bool |  | false |  |  |
| `--generate_audio` | bool |  | false |  |  |
| `--seed` | int |  | -1 |  |  |

```bash
bara generate create seedance_1_5_pro_i2v --prompt "..." --wait
```

### sora_2_i2v

**SORA 2 I2V** — video via runpod

- Endpoint: `sora-2-i2v`
- Price: $0.40 (4s) / $0.80 (8s) / $1.20 (12s)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--duration` | int | yes |  | `4`, `8`, `12` |  |

```bash
bara generate create sora_2_i2v --prompt "..." --wait
```

### sora_2_pro_i2v

**SORA 2 Pro I2V** — video via runpod

- Endpoint: `sora-2-pro-i2v`
- Price: $1.20 - $6.00 per video (by resolution and duration)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--size` | string |  | 720p | `720p`, `1080p` |  |
| `--duration` | int | yes |  | `4`, `8`, `12` |  |

```bash
bara generate create sora_2_pro_i2v --prompt "..." --wait
```

### vidu_q3_i2v

**Vidu Q3 I2V** — video via runpod

- Endpoint: `vidu-q3-i2v`
- Price: $0.15 per second

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--resolution` | string |  | 720p | `540p`, `720p`, `1080p` |  |
| `--duration` | int |  | 5 | 1–16 |  |
| `--movement_amplitude` | string |  | auto | `auto`, `small`, `medium`, `large` |  |
| `--generate_audio` | bool |  | true |  |  |
| `--bgm` | bool |  | true |  |  |
| `--seed` | int |  | -1 |  |  |

```bash
bara generate create vidu_q3_i2v --prompt "..." --wait
```

### vidu_q3_t2v

**Vidu Q3 T2V** — video via runpod

- Endpoint: `vidu-q3-t2v`
- Price: $0.15 per second

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--style` | string |  | general | `general`, `anime` |  |
| `--resolution` | string |  | 720p | `540p`, `720p`, `1080p` |  |
| `--duration` | int |  | 5 | 1–16 |  |
| `--aspect_ratio` | string |  | 4:3 | `16:9`, `9:16`, `4:3`, `3:4`, `1:1` |  |
| `--movement_amplitude` | string |  | auto | `auto`, `small`, `medium`, `large` |  |
| `--generate_audio` | bool |  | true |  |  |
| `--bgm` | bool |  | true |  |  |
| `--seed` | int |  | -1 |  |  |

```bash
bara generate create vidu_q3_t2v --prompt "..." --wait
```

### wan_2_1_i2v

**WAN 2.1 I2V 720p** — video via runpod

- Endpoint: `wan-2-1-i2v-720`
- Price: $0.30 per 5 seconds

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--negative_prompt` | string |  |  |  |  |
| `--size` | string |  | 1280*720 |  |  |
| `--num_inference_steps` | int |  | 30 | 1–50 |  |
| `--guidance` | float |  | 5 | 0–10 |  |
| `--duration` | int |  | 5 |  |  |
| `--flow_shift` | int |  | 5 |  |  |
| `--seed` | int |  | -1 |  |  |
| `--enable_prompt_optimization` | bool |  | false |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create wan_2_1_i2v --prompt "..." --wait
```

### wan_2_2_i2v

**WAN 2.2 I2V 720p** — video via runpod

- Endpoint: `wan-2-2-i2v-720`
- Price: $0.30 - $0.90 per video (5-15s)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--negative_prompt` | string |  |  |  |  |
| `--size` | string |  | 1280*720 |  |  |
| `--num_inference_steps` | int |  | 30 | 1–50 |  |
| `--guidance` | float |  | 5 | 0–10 |  |
| `--duration` | int |  | 5 |  |  |
| `--flow_shift` | int |  | 5 |  |  |
| `--seed` | int |  | -1 |  |  |
| `--enable_prompt_optimization` | bool |  | false |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create wan_2_2_i2v --prompt "..." --wait
```

### wan_2_2_i2v_lora

**WAN 2.2 I2V 720p LoRA** — video via runpod

- Endpoint: `wan-2-2-t2v-720-lora`
- Price: $0.35 (5s) / $0.56 (8s)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--high_noise_loras` | json |  |  |  | [{path, scale}] |
| `--low_noise_loras` | json |  |  |  | [{path, scale}] |
| `--duration` | int |  | 5 |  |  |
| `--seed` | int |  | -1 |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create wan_2_2_i2v_lora --prompt "..." --wait
```

### wan_2_5

**WAN 2.5 I2V** — video via runpod

- Endpoint: `wan-2-5`
- Price: $0.25 - $1.50 per video (480p/5s to 1080p/10s)

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--negative_prompt` | string |  |  |  |  |
| `--size` | string |  | 1280*720 |  | width*height |
| `--duration` | int |  | 5 |  | seconds |
| `--seed` | int |  | -1 |  |  |
| `--enable_prompt_expansion` | bool |  | false |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create wan_2_5 --prompt "..." --wait
```

### wan_2_6_i2v

**WAN 2.6 I2V** — video via runpod

- Endpoint: `wan-2-6-i2v`
- Price: 720p $0.10/sec, 1080p $0.15/sec

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--image` | string | yes |  | image path or URL |  |
| `--audio` | string |  |  | audio path or URL |  |
| `--negative_prompt` | string |  |  |  |  |
| `--size` | string |  | 1280*720 | `1280*720`, `1920*1080` |  |
| `--duration` | int |  | 5 | `5`, `10`, `15` |  |
| `--shot_type` | string |  | single | `single`, `multi` |  |
| `--seed` | int |  | -1 |  |  |
| `--enable_prompt_expansion` | bool |  | false |  |  |
| `--enable_safety_checker` | bool |  | true |  |  |

```bash
bara generate create wan_2_6_i2v --prompt "..." --wait
```

### wan_2_6_t2v

**WAN 2.6 T2V** — video via runpod

- Endpoint: `wan-2-6-t2v`
- Price: 720p $0.50-$1.50 / 1080p $0.75-$2.25 by duration

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--negative_prompt` | string |  |  |  |  |
| `--duration` | int |  | 5 | `5`, `10`, `15` |  |
| `--size` | string |  | 1280*720 | `1280*720`, `1920*1080`, `720*1280`, `1080*1920` |  |
| `--shot_type` | string |  | single | `single`, `multi` |  |
| `--seed` | int |  | -1 |  |  |
| `--enable_prompt_expansion` | bool |  | false |  |  |
| `--audio` | string |  |  | audio path or URL | optional audio track URL |

```bash
bara generate create wan_2_6_t2v --prompt "..." --wait
```

### chatterbox_turbo

**Chatterbox Turbo** — audio via runpod

- Endpoint: `chatterbox-turbo`
- Price: $0.001 per second of audio

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  | text to speak |
| `--voice` | string |  | lucy | `aaron`, `abigail`, `anaya`, `andy`, `archer`, `brian`, `chloe`, `dylan`, `emmanuel`, `ethan`, `evelyn`, `gavin`, `gordon`, `ivan`, `laura`, `lucy`, `madison`, `marisol`, `meera`, `walter` |  |
| `--voice_url` | string |  |  | audio path or URL | reference clip for voice cloning |
| `--format` | string |  | wav | `wav`, `flac`, `ogg` |  |

```bash
bara generate create chatterbox_turbo --prompt "..." --wait
```

### minimax_speech_02_hd

**Minimax Speech 02 HD** — audio via runpod

- Endpoint: `minimax-speech-02-hd`
- Price: $0.05 per 1000 characters

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  | text to speak |
| `--voice_id` | string |  | Wise_Woman |  |  |
| `--speed` | float |  | 1 |  |  |
| `--volume` | float |  | 1 |  |  |
| `--pitch` | float |  | 0 |  |  |
| `--emotion` | string |  | neutral | `happy`, `sad`, `neutral`, `angry`, `fearful`, `surprised` |  |
| `--english_normalization` | bool |  | false |  |  |
| `--default_audio_url` | string |  |  | audio path or URL |  |

```bash
bara generate create minimax_speech_02_hd --prompt "..." --wait
```

### granite_4

**IBM Granite 4.0 H Small** — text via runpod

- Endpoint: `granite-4-0-h-small`
- Price: $10.00 per 1M tokens

Chat-shaped: --prompt is wrapped into a single user message. Use --messages for full control.

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string |  |  |  | shorthand for a single user message |
| `--messages` | json |  |  |  | [{role, content}] |
| `--max_tokens` | int |  | 512 |  |  |
| `--temperature` | float |  | 0.7 |  |  |
| `--seed` | int |  | -1 |  |  |
| `--top_k` | int |  | -1 |  |  |
| `--top_p` | float |  | 1 |  |  |

```bash
bara generate create granite_4 --prompt "..." --wait
```

### qwen3_32b

**Qwen3 32B AWQ** — text via runpod

- Endpoint: `qwen3-32b-awq`
- Price: $10.00 per 1M tokens

Also exposes an OpenAI-compatible base at https://api.runpod.ai/v2/qwen3-32b-awq/openai/v1

| parameter | type | required | default | allowed | description |
|---|---|---|---|---|---|
| `--prompt` | string | yes |  |  |  |
| `--max_tokens` | int |  | 512 |  |  |
| `--temperature` | float |  | 0.7 |  |  |
| `--top_p` | float |  |  |  |  |
| `--top_k` | int |  |  |  |  |
| `--stop` | string |  |  |  |  |

```bash
bara generate create qwen3_32b --prompt "..." --wait
```

