# OneDay Model Benchmark

Questo benchmark misura i modelli su task realistici di `oneday`, non su prompt generici.

## Obiettivo

Confrontare modelli LLM su due assi:

- compatibilita' col motore del gioco
- qualita' narrativa percepita sui prompt reali di OneDay

## Casi Coperti

1. `story-creation-final`
Generazione del JSON finale di creazione storia.

2. `narrative-intro`
Primo turno narrativo con contratto `NarrativeResponse`.

3. `dialogue-metadata`
Scena dialogata che dovrebbe produrre `dialogue_blocks`, `entities_mentioned`, `event_callouts` e choice metadata.

4. `challenge-scene`
Scena d'azione che dovrebbe produrre almeno una `challenge` valida senza risolverla da sola.

5. `chapter-summary`
Titolo + riassunto di capitolo in JSON.

## Metriche Automatiche

- validita' del payload JSON, sia fenced che raw
- aderenza al tipo Go previsto
- campi richiesti presenti
- conteggi corretti per choices / liste / word count
- completion time totale
- prompt tokens / completion tokens
- throughput stimato in token/secondo e caratteri/secondo
- stima costo USD
- context window / max completion dal catalogo modelli OpenRouter

## Modalita'

Il benchmark supporta tre modalita':

- `prompt`: nessun `response_format`, solo prompt + parser tollerante
- `json_object`: `response_format: {type: json_object}`
- `json_schema`: `response_format: {type: json_schema, ...}` con schema per caso

Per confrontare bene i modelli conviene eseguire tutte e tre.

## Qualita' Narrativa

La qualita' narrativa non viene decisa automaticamente dal tool.
Va letta e giudicata a mano sui report generati, con attenzione a:

- atmosfera e voce
- chiarezza della scena
- agency del player
- qualita' delle choices
- coerenza con il tono di OneDay
- uso sensato dei metadata semantici

## Esecuzione

Esempio con OpenRouter diretto:

```bash
OPENROUTER_API_KEY=... \
go run ./cmd/oneday-benchmark \
  -base-url https://openrouter.ai/api/v1 \
  -mode all \
  -models x-ai/grok-4.1-fast,qwen/qwen3.5-flash-02-23,google/gemini-2.5-flash-lite,google/gemini-3.1-flash-lite-preview
```

Output:

- report JSON in `docs/benchmarks/runs/`
- report Markdown in `docs/benchmarks/runs/`
