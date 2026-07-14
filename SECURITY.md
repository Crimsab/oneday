# Security policy

## Supported versions

Security fixes target the latest released `1.x` version and the `main` branch.
Older releases may receive a fix only when the change can be backported safely.

## Report a vulnerability

Do not open a public issue for a vulnerability. Use a
[private GitHub security advisory](https://github.com/Crimsab/oneday/security/advisories/new)
and include:

- affected version or commit;
- impact and realistic attack path;
- the smallest safe reproduction;
- any suggested mitigation.

Remove API keys, authentication files, database content, private prompts, and
story text from reports. You should receive an initial response within seven
days. A fix and disclosure timeline will be coordinated according to severity.

## Scope

OneDay is designed for local and self-hosted use. Exposing the gateway directly
to an untrusted network requires deployment controls outside this repository,
including authentication, TLS, network policy, backups, and provider-key
protection.
