# Release Notes

## Unreleased

### Breaking Changes

#### Demo Mode Proxy Allowlist

Demo mode session IP handling now requires an explicit configuration choice.

When `demo.enabled` is `true`, configure one of these modes:

- Set `demo.allowed_proxies` to use Echo's built-in `X-Forwarded-For` handling with those IPs or CIDRs added as trusted ranges.
- Set `demo.disable_proxy_headers: true` to ignore forwarded headers and use `RemoteAddr` only.

`demo.allowed_proxies` accepts exact IPs and CIDRs. `X-Real-IP` is no longer used for demo session IP pinning.

This replaces the earlier demo proxy config names:

- `demo.trust_proxy_headers`
- `demo.trusted_proxies`

Setting both `demo.allowed_proxies` and `demo.disable_proxy_headers: true` is invalid. Upstream proxies must overwrite `X-Forwarded-For` rather than passing through user-supplied values.
