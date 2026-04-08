# Align OneDay LiteLLM Virtual Key For RAG

**Captured:** 2026-04-09
**Status:** Pending
**Type:** Ops / proxy configuration

## Problem

OneDay currently attempts to generate embeddings through LiteLLM using `text-embedding-3-small`, but the active virtual key used by the app does not have access to that embedding model. This causes RAG embedding calls to fail with `key_model_access_denied`.

## Why this is not a repo-only fix

The application code is correct to call the configured embedding model. The missing piece is the upstream LiteLLM virtual key scope, which lives outside this repository.

## Desired outcome

Either:

1. Update the OneDay virtual key to include `text-embedding-3-small` (preferred), or
2. Temporarily disable RAG/embeddings for OneDay until the key policy is fixed

## References

- `/workspace/homelab/docs/ai-proxy.md`
- `/opt/lab/docker/ai-proxy/config.yaml`
- `/opt/lab/docker/oneday/config.yaml`

