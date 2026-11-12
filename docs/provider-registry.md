# Provider registry and model routing (PR #15)

Set `PLOOS_VIDEO_PROVIDERS_JSON` to a JSON array of provider definitions. Example (set credentials separately in environment variables):

```json
[
 {"name":"local","kind":"async","endpoint":"http://127.0.0.1:9001/jobs","token_env":"PLOOS_LOCAL_VIDEO_TOKEN","models":["local-video-v1"],"local":true},
 {"name":"secondary","kind":"http","endpoint":"http://127.0.0.1:9002/generate","token_env":"PLOOS_SECONDARY_VIDEO_TOKEN","models":["secondary-video-v1"],"local":true}
]
```

The registry is opt-in; legacy `PLOOS_VIDEO_ASYNC_ENDPOINT` and `PLOOS_VIDEO_HTTP_ENDPOINT` remain supported in single-provider mode. Do not combine modes. Providers are still restricted to private/loopback IP endpoints; this does not enable direct public paid APIs.

The authenticated MCP `video_models` tool lists configured models with provider names and the operator-declared `local` flag. This flag is descriptive only and **not** proof of locality, cost, privacy or security. The registry requires each model ID to map to exactly one provider. Unknown model IDs are rejected before a job is submitted. No automatic fallback or paid provider selection occurs.

**Recovery caution:** Model IDs must retain the same upstream provider mapping across restarts while remote jobs remain active. Changing mappings can prevent recovery or poll the wrong upstream provider. Future work should persist immutable provider identities alongside remote job IDs and verify them before resuming.

Idempotency keys continue to be forwarded by providers that implement them. A provider must enforce idempotency upstream to prevent duplicate charges.
