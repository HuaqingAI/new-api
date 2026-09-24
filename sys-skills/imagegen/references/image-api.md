# new-api Images API quick reference

This skill calls new-api through OpenAI-compatible image endpoints.

## Endpoints

Generation:

```http
POST {base_url}/images/generations
Authorization: Bearer <api_key>
Content-Type: application/json
```

Edit:

```http
POST {base_url}/images/edits
Authorization: Bearer <api_key>
Content-Type: multipart/form-data
```

`base_url` may be provided with or without `/v1`; the CLI normalizes it to a
single `/v1` suffix and avoids `/v1/v1`.

Examples:

- `https://hth.huaqing.run` -> `https://hth.huaqing.run/v1`
- `https://hth.huaqing.run/v1/` -> `https://hth.huaqing.run/v1`
- `https://hth.huaqing.run/v1/images/generations` -> `https://hth.huaqing.run/v1`

## Fixed Model

Every request sends:

```json
{ "model": "gpt-image-2" }
```

OpenCode/Codex model config may identify a provider, but it must not replace the
image model.

## Generation Body

Minimum:

```json
{
  "model": "gpt-image-2",
  "prompt": "a cat",
  "size": "1024x1024"
}
```

Typical CLI body:

```json
{
  "model": "gpt-image-2",
  "prompt": "a cat",
  "n": 1,
  "size": "1024x1024",
  "quality": "medium",
  "output_format": "png",
  "response_format": "b64_json"
}
```

Supported request fields:

- `prompt`
- `model` fixed to `gpt-image-2`
- `n` from `1` to `10`
- `size` as `auto` or constrained `WIDTHxHEIGHT`
- `quality`: `low`, `medium`, `high`, `auto`
- `output_format`: `png`, `jpeg`, `webp`
- `output_compression`: `0..100`
- `response_format`: `b64_json` or `url`
- `moderation`
- `background`: `opaque` or `auto`; `transparent` is rejected by this fixed-model skill

## Edit Multipart Fields

Text fields:

```text
model=gpt-image-2
prompt=turn the cat into a watercolor poster
n=1
size=1024x1024
quality=medium
output_format=png
response_format=b64_json
```

File fields:

```text
image=@output/imagegen/cat.png
mask=@input/mask.png
```

For multiple input images, the CLI sends repeated `image[]` fields. new-api's
OpenAI adapter accepts `image`, `image[]`, and `image[index]`.

`mask` is optional and singular.

## Response

The CLI expects an OpenAI-compatible image response:

```json
{
  "created": 1789370877,
  "data": [
    {
      "b64_json": "iVBO...",
      "revised_prompt": "a cat"
    }
  ],
  "background": "auto",
  "output_format": "png",
  "quality": "auto",
  "size": "1254x1254",
  "model": "gpt-image-2-codex",
  "usage": {
    "input_tokens": 8,
    "output_tokens": 515,
    "total_tokens": 523
  }
}
```

Parsing rules:

- Prefer `data[].b64_json`; decode and write local image files.
- If only `data[].url` is returned, download the URL and still write a local
  file. Do not leave remote URLs as final assets.
- Preserve `revised_prompt`, response `model`, `size`, `quality`,
  `output_format`, `created`, and `usage` in `.imagegen-state.json`.
- Never print the raw base64 image in stdout.

## AionUi Rendering

The CLI prints image metadata that AionUi can render:

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

Final assistant messages should include a Markdown image link:

```markdown
![generated image](output/imagegen/cat.png)
```

AionUi resolves relative Markdown image paths against the conversation
workspace, so generated images should stay under `output/imagegen/`.

