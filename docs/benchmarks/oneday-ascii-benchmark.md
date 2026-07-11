# OneDay ASCII Benchmark

Benchmark dedicato ai modelli che potrebbero generare ASCII art ambientale per `oneday`.

## Obiettivo

Confrontare modelli specialistici o economici per:

- tempo totale di generazione
- throughput
- costo stimato
- contesto disponibile
- aderenza ai vincoli TUI (`<= 12` linee, `<= 72` colonne)
- utilita' pratica per location reveal, signage, terminali, mappe, rituali e oggetti iconici

## Casi Coperti

1. `location-reveal`
2. `neon-signage`
3. `terminal-screen`
4. `ritual-circle`
5. `map-fragment`
6. `artifact-reveal`

## Modalita'

Supporta:

- `prompt`
- `json_object`
- `json_schema`
- `all`

Per il runtime reale di OneDay, `json_schema` e' la modalita' piu' rappresentativa.

## Esecuzione

```bash
OPENROUTER_API_KEY=... \
go run ./cmd/oneday-ascii-benchmark \
  -base-url https://openrouter.ai/api/v1 \
  -mode json_schema
```

Output:

- report JSON in `docs/benchmarks/runs/`
- report Markdown in `docs/benchmarks/runs/`
