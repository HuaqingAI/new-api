---
name: "imagegen"
description: "Generate or edit raster images through the configured new-api OpenAI-compatible Images API. Use when Codex should create a bitmap image, edit an existing bitmap, or continue editing a previously generated image. Do not use for SVG/vector/code-native assets, and do not use AionUi's built-in image MCP."
---

# Image Generation Skill

This fork generates and edits images by running `scripts/image_gen.py` directly
against a new-api OpenAI-compatible Images API.

## Hard Rules

- Do not use the built-in `image_gen` tool for this skill.
- Do not use AionUi's built-in image MCP.
- Always call the local CLI: `scripts/image_gen.py`.
- The image model is fixed to `gpt-image-2`.
- Do not inherit the Codex/OpenCode chat model as the image model.
- Do not pass another model unless the user asks to change this skill's code.
- Base URL and API key are discovered from OpenCode/Codex config first, then may
  be overridden by environment variables or CLI arguments.
- Save generated assets in the current conversation workspace, normally under
  `output/imagegen/`.
- Never leave the only useful image copy in the skill directory, a temp folder,
  or `$CODEX_HOME/generated_images`.
- Do not print API keys, full auth config, or inline image base64.
- For AionUi rendering, report the saved image path and include a Markdown image
  link such as `![generated](output/imagegen/name.png)`.

## CLI Surface

The CLI supports:

- `generate`
- `edit`
- `generate-batch`

Primary commands:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" generate \
  --prompt "a cat" \
  --size 1024x1024
```

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" edit \
  --from-last \
  --prompt "turn the image into a watercolor poster"
```

Explicit new-api override:

```bash
python "F:/skill项目/GPT生图/imagegen/scripts/image_gen.py" generate \
  --base-url "https://hth.huaqing.run/v1" \
  --api-key "$IMAGEGEN_API_KEY" \
  --prompt "a cat" \
  --size 1024x1024
```

## Config Resolution

The CLI reads base URL and key in this order:

1. OpenCode config:
   - `$OPENCODE_CONFIG_DIR/opencode.jsonc`
   - `opencode.jsonc` in the workspace
   - `.opencode/opencode.jsonc` in the workspace
2. Codex config:
   - `.codex/config.toml` and `.codex/auth.json` in the workspace
   - `$CODEX_HOME/config.toml` and `$CODEX_HOME/auth.json`
   - `~/.codex/config.toml` and `~/.codex/auth.json`
3. OpenAI-compatible environment overrides:
   - `OPENAI_BASE_URL` or `OPENAI_API_BASE`
   - `OPENAI_API_KEY`
4. imagegen-specific environment overrides:
   - `IMAGEGEN_BASE_URL`
   - `IMAGEGEN_API_KEY`
   - `IMAGEGEN_PROVIDER`
5. CLI overrides:
   - `--base-url`
   - `--api-key`
   - `--provider`

The OpenCode/Codex model field may be used only to locate a provider, for
example `hth/...`; image requests still send `model: gpt-image-2`.

## API Contract

Generation:

```http
POST {base_url}/images/generations
Authorization: Bearer <api_key>
Content-Type: application/json
```

The request body always includes `model: gpt-image-2`, plus the prompt, size,
quality, output format, `n`, and other supported options.

Editing:

```http
POST {base_url}/images/edits
Authorization: Bearer <api_key>
Content-Type: multipart/form-data
```

Use `--image <path>` for explicit image files. Use `--from-last` or
`--image last` to continue editing the latest image recorded in
`output/imagegen/.imagegen-state.json`.

For multiple input images, pass repeated `--image` flags and describe their
roles in the prompt. A single optional `--mask` is supported.

## Output Contract

The CLI writes binary image files and prints JSON to stdout. Human progress logs
go to stderr.

The JSON includes AionUi-compatible image metadata:

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

When reporting the result to the user, include the relative path and a Markdown
image preview:

```markdown
![generated image](output/imagegen/cat.png)
```

## Workflow

1. Decide whether the user wants a new image (`generate`) or a modification of
   an existing image (`edit`).
2. Shape the prompt with the shared prompt schema below when it improves the
   output. Do not add arbitrary objects, brands, text, or narrative details.
3. Run `scripts/image_gen.py` from the current workspace.
4. Use `--out` only when a stable filename is needed; otherwise allow the CLI to
   create a timestamped file under `output/imagegen/`.
5. For edits, use explicit `--image` paths when supplied. Use `--from-last` only
   when the user clearly wants to continue from the previous generated result.
6. Verify the CLI completed and the output file exists.
7. Reply with the saved path, the prompt used, and a Markdown image link.

## Prompt Schema

Use only the useful lines:

```text
Use case: <photorealistic-natural|product-mockup|ui-mockup|infographic-diagram|scientific-educational|ads-marketing|productivity-visual|logo-brand|illustration-story|stylized-concept|historical-scene|text-localization|identity-preserve|precise-object-edit|lighting-weather|background-extraction|style-transfer|compositing|sketch-to-render>
Asset type: <where the asset will be used>
Primary request: <user's main prompt>
Input images: <Image 1: role; Image 2: role>
Scene/backdrop: <environment>
Subject: <main subject>
Style/medium: <photo/illustration/3D/etc>
Composition/framing: <wide/close/top-down; placement>
Lighting/mood: <lighting + mood>
Color palette: <palette notes>
Materials/textures: <surface details>
Text (verbatim): "<exact text>"
Constraints: <must keep/must avoid>
Avoid: <negative constraints>
```

For edits, repeat invariants clearly, for example: `change only the background;
keep the subject, pose, edges, and text unchanged`.

## Model Notes

`gpt-image-2` supports:

- `quality`: `low`, `medium`, `high`, `auto`
- `size`: `auto` or `WIDTHxHEIGHT` when all constraints hold:
  - max edge `<= 3840px`
  - both edges are multiples of `16px`
  - long-to-short ratio `<= 3:1`
  - total pixels from `655,360` to `8,294,400`

Common sizes:

- `1024x1024`
- `1536x1024`
- `1024x1536`
- `2048x2048`
- `2048x1152`
- `3840x2160`
- `2160x3840`
- `auto`

This fixed-model skill does not support `background=transparent` or
`input_fidelity`; do not switch to another model without an explicit code-change
request.

## References

- `references/cli.md`: CLI usage and examples.
- `references/image-api.md`: new-api/OpenAI-compatible Images API behavior.
- `references/codex-network.md`: network and sandbox troubleshooting.
- `references/prompting.md`: shared prompt writing principles.
- `references/sample-prompts.md`: reusable prompt recipes.
- `scripts/image_gen.py`: CLI implementation.
