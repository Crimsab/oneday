# Summary 37.2 - Post-commit queue and provider boundary

Only assistant messages with an immutable source commit on the active branch can
be segmented or queued. Structured dialogue retains canonical speaker IDs;
narration and dialogue keep source order; provisional messages produce no
segments. Global story policy and per-assignment policy run before cache lookup.

Cache identity covers provider, model, voice, version, language, normalized
canonical text, canonical style JSON, speed, format, and pronunciation revision.
Queue insertion is idempotent. Ready regular cache files are reused; stale or
missing files are not trusted.

Cloud Speech and persistent Piper HTTP adapters implement the same provider
contract. Disabled, incomplete, or unreachable providers fail as audio jobs and
never roll back a story turn. Successful work is atomically written to the audio
cache and creates a `tts` generation run causally parented to the narrator run.

Go 532 passed across 24 packages. Live deployment on `homelab-main` remains
schema V35 with integrity `ok`, health `ok`, restart count zero, and zero audio
assets/jobs/profiles because both existing stories retain the safe default
policy `off`. Backup: `/opt/lab/backups/oneday/phase37-2-20260711`.
