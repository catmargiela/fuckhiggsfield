# @catmargiela/bara

Generate images with GPT Image 2 from the terminal using your own OpenAI API
key, and get every result back as a catbox.moe link.

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
bara model get gpt_image_2
bara generate create gpt_image_2 --prompt "a cat in a spaceship" --wait
bara --help
```

Full documentation: <https://github.com/catmargiela/fuckhiggsfield>

## License

MIT
