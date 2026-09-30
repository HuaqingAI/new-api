# CLI reference (`scripts/image_gen.py`)

This CLI is the primary execution path for this `imagegen` skill fork. It calls
new-api's OpenAI-compatible Images API directly. It does not use the official
Codex built-in `image_gen` tool and does not use AionUi's built-in image MCP.

## Fixed Model

All requests send:

```json
{ "model": "gpt-image-2" }
```

The CLI accepts legacy `--model gpt-image-2` as a no-op for compatibility, but
any other model fails. Do not use `IMAGEGEN_MODEL` or OpenCode/Codex chat model
values as the image model.

## Config

Base URL and API key are discovered from OpenCode/Codex config first, then can
be overridden by env vars or CLI args.

Discovery and override sources:

- OpenCode: `$OPENCODE_CONFIG_DIR/opencode.jsonc`, `opencode.jsonc`,
  `.opencode/opencode.jsonc`
- Codex: `.codex/config.toml`, `.codex/auth.json`, `$CODEX_HOME/config.toml`,
  `$CODEX_HOME/auth.json`, `~/.codex/config.toml`, `~/.codex/auth.json`
- Env override: `OPENAI_BASE_URL`, `OPENAI_API_BASE`, `OPENAI_API_KEY`
- Env override: `IMAGEGEN_BASE_URL`, `IMAGEGEN_API_KEY`, `IMAGEGEN_PROVIDER`
- CLI override: `--base-url`, `--api-key`, `--provider`

Never print or paste full API keys in chat.

## Commands

Dry run:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" generate \
  --prompt "a cat" \
  --size 1024x1024 \
  --dry-run \
  --no-augment
```

Generate:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" generate \
  --prompt "a cat" \
  --size 1024x1024
```

Generate with explicit new-api config:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" generate \
  --base-url "https://hth.huaqing.run/v1" \
  --api-key "$IMAGEGEN_API_KEY" \
  --prompt "a cat" \
  --size 1024x1024
```

Edit an explicit image:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" edit \
  --image output/imagegen/cat.png \
  --prompt "make the cat wear a tiny red scarf; keep the same pose"
```

Continue from latest generated image:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" edit \
  --from-last \
  --prompt "turn the image into a watercolor poster"
```

Equivalent latest-image form:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" edit \
  --image last \
  --prompt "make it a night scene"
```

Edit with a mask:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" edit \
  --image output/imagegen/cat.png \
  --mask input/mask.png \
  --prompt "replace only the background; keep the cat unchanged"
```

Batch generation from JSONL:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" generate-batch \
  --input tmp/imagegen/prompts.jsonl \
  --out-dir output/imagegen/batch \
  --concurrency 5
```

JSONL lines can be strings or objects:

```jsonl
{"prompt":"A product thumbnail of a matte ceramic mug","size":"1024x1024","quality":"low","out":"mug.png"}
{"prompt":"A polished landing-page hero image of the same mug","size":"2048x1152","quality":"high","out":"mug-hero.png"}
```

Per-job `model` is ignored only if it is `gpt-image-2`; any other model fails.

## Output

If `--out` is omitted, the CLI writes a timestamped file under:

```text
output/imagegen/
```

It also updates:

```text
output/imagegen/.imagegen-state.json
```

Stdout is JSON and includes AionUi-compatible metadata:

```json
{
  "saved_path": "E:\\workspace\\output\\imagegen\\cat.png",
  "image": {
    "path": "E:\\workspace\\output\\imagegen\\cat.png",
    "mime_type": "image/png",
    "source": "codex_image_generation"
  },
  "result_omitted": true,
  "result_omitted_reason": "image_base64"
}
```

Human progress messages are written to stderr.

## Common Options

- `--prompt` / `--prompt-file`
- `--size auto|WIDTHxHEIGHT`
- `--quality low|medium|high|auto`
- `--n 1..10`
- `--output-format png|jpeg|webp`
- `--output-compression 0..100`
- `--response-format b64_json|url`
- `--out <path>`
- `--out-dir <dir>`
- `--workspace <path>`
- `--state-file <path>`
- `--timeout <seconds>`
- `--http-retries 1..10`
- `--force`
- `--dry-run`
- `--no-augment`

Edit-only:

- `--image <path>` (repeatable)
- `--image last`
- `--from-last`
- `--mask <png>`

This fixed-model skill rejects:

- `--background transparent`
- `--input-fidelity`
- any model other than `gpt-image-2`

