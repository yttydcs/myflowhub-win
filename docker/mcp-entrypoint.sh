#!/bin/sh
set -eu

if [ -z "${MYFLOWHUB_MCP_AUTH_TOKEN:-}" ]; then
  echo "MYFLOWHUB_MCP_AUTH_TOKEN is required for remote MCP HTTP mode." >&2
  exit 1
fi

set -- \
  /usr/local/bin/myflowhub-mcp \
  --transport http \
  --allow-remote \
  --listen "${MYFLOWHUB_MCP_LISTEN:-0.0.0.0:17688}" \
  --mcp-path "${MYFLOWHUB_MCP_PATH:-/mcp}" \
  --config-dir "${MYFLOWHUB_MCP_CONFIG_DIR:-/data}" \
  --device-id "${MYFLOWHUB_MCP_DEVICE_ID:-ai-node}" \
  --display-name "${MYFLOWHUB_MCP_DISPLAY_NAME:-AI MCP}"

if [ -n "${MYFLOWHUB_MCP_ENDPOINT:-}" ]; then
  set -- "$@" --endpoint "$MYFLOWHUB_MCP_ENDPOINT"
fi

if [ -n "${MYFLOWHUB_MCP_DEFAULT_TARGET:-}" ]; then
  set -- "$@" --default-target "$MYFLOWHUB_MCP_DEFAULT_TARGET"
fi

if [ -n "${MYFLOWHUB_MCP_TIMEOUT:-}" ]; then
  set -- "$@" --timeout "$MYFLOWHUB_MCP_TIMEOUT"
fi

case "${MYFLOWHUB_MCP_ALLOW_WRITE:-false}" in
  1|true|TRUE|True|yes|YES|Yes|on|ON|On)
    set -- "$@" --allow-write
    ;;
esac

exec "$@"
