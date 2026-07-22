# @catmargiela/bara

Generate images, video, audio, and text from the terminal using your own OpenAI
and RunPod API keys.

## Install

```bash
npm install -g @catmargiela/bara
```

The postinstall step downloads the prebuilt `bara` binary for your platform from
[GitHub Releases](https://github.com/catmargiela/fuckhiggsfield/releases) and
verifies it against the release `checksums.txt`.

Supported: macOS, Linux, Windows on x64 and arm64.

## Usage

```bash
bara settings set openai_api_key --stdin
bara model list --image
bara generate create gpt_image_2 --prompt "a cat in a spaceship" --wait
bara --help
```

Full documentation: <https://github.com/catmargiela/fuckhiggsfield>

## License

MIT
