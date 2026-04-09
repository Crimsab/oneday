# Plan 18.2 Summary

Added a structured `social_duel` narrator/runtime contract to gameplay responses, including prompt guidance, JSON schema support, persistence through stored assistant metadata, and normalization for partial provider output.

This keeps AI-authored duel framing separate from engine-owned mechanics while making offer/continue metadata survive resume/history and degrade safely when fields are missing or malformed.

Code commit: `77ae07a feat(social-duel): add narrator contract metadata`
