# OneDay Model Benchmark

- Generated: `2026-04-08T22:15:32+02:00`
- Base URL: `https://openrouter.ai/api/v1`
- Mode: `prompt`
- Scoring: `compatibility only` from the command output. Narrative quality is reviewed separately by hand.
- Raw JSON artifact: `2026-04-08-221946-oneday-benchmark-prompt.json`

## Leaderboard

| Model | Compat Score | Success Rate | Avg Seconds | Avg Cost | Context | Max Out | Avg Tok/s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `google/gemini-3.1-flash-lite-preview` | 94.5 | 100% | 8.497 | $0.001711 | 1048576 | 65536 | 74.9 |
| `x-ai/grok-4.1-fast` | 94.0 | 100% | 0.723 | $0.001399 | 2000000 | 30000 | 2235.1 |
| `qwen/qwen3.5-flash-02-23` | 94.0 | 100% | 0.680 | $0.000803 | 1000000 | 65536 | 4141.7 |
| `google/gemini-2.5-flash-lite` | 92.2 | 100% | 0.617 | $0.000592 | 1048576 | 65535 | 1137.0 |

## google/gemini-3.1-flash-lite-preview

- Context: `1048576`
- Max completion: `65536`
- Prompt cost/token: `$0.000000250`
- Completion cost/token: `$0.000001500`
- Avg latency: `8.497 s`
- Avg cost: `$0.001711`
- Total benchmark cost: `$0.008555`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `9.224 s`
- Completion tokens: `835`
- Prompt tokens: `1005`
- Estimated cost: `$0.001504`
- Throughput: `90.5 tok/s`, `282.8 char/s`

```text
{
  "name": "Vespera: L'Eco delle Campane",
  "description": "Una Venezia decadente e oscura costruita su isole nere, dove il potere si misura in segreti e il mare sussurra promesse proibite. In questo mondo malinconico, le reliquie sommerse e le campane sommerse dettano il ritmo di una politica spietata e di culti misteriosi.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico, politico, avventuroso",
  "setting": {
    "world_name": "Vespera",
    "era": "L'Era dell'Eclissi Salina",
    "geography": "Un labirinto di canali di sale, isole di pietra nera e quartieri parzialmente sommersi pieni di reliquie antiche.",
    "magic_system": "Magia sonora basata sulle campane sommerse; ogni suono ha il potere di alterare la realtà, ma richiede un sacrificio di memoria o salute mentale.",
    "technology_level": "Rinascimentale decadente con elementi magici.",
    "society": "Una gerarchia rigida dominata da corporazioni mercantili e nobili di canale, dove il debito è una condanna legale e la corruzione è la norma.",
    "rules": [
      "La magia ha sempre un costo fisico o mentale.",
      "Il sale protegge dai sussurri del mare.",
      "Le campane sommerse alterano memoria e volontà.",
      "I giuramenti pubblici hanno valore legale e spirituale.",
      "Il debito non pagato può essere venduto come servitù."
    ],
    "factions": [
      "Casata Valcerra (Corporazione Mercantile)",
      "Coro del Sale (Religione delle Campane)",
      "Guardie di Marea (Forze dell'ordine corrotte)"
    ],
    "cultures": [
      "Nobili di Canale",
      "Scavatori di Relitti",
      "Pellegrini delle Campane"
    ],
    "dangers": [
      "Nebbie senzienti",
      "Reliquie infette",
      "Inquisitori del suono",
      "Allagamenti improvvisi"
    ]
  },
  "stats_schema": {
 
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `10.080 s`
- Completion tokens: `646`
- Prompt tokens: `4409`
- Estimated cost: `$0.002071`
- Throughput: `64.1 tok/s`, `213.8 char/s`

```text
{
  "narrative": "La nebbia avvolge la Banchina delle Nebbie come un sudario umido, intriso di sale acre e dell'odore metallico dei canali stagnanti. Sei ferma sul bordo di un molo di pietra consunta; sotto di te, l'acqua nera della laguna sciaborda ritmicamente contro i pali di legno marcio, un battito cardiaco lento che sembra scandire il tempo di Vespera. In lontananza, un rintocco sordo e profondo risuona dal cuore della città: una delle campane sommerse ha appena suonato, e senti una vibrazione familiare risalirti lungo la spina dorsale, un promemoria che qui, in questa città di relitti, ogni respiro ha un costo.\n\nHai poche Corone di sale in tasca e il peso dei debiti che gravano sulla tua famiglia ti schiaccia più dell'umidità dell'aria. Davanti a te, un battello di fortuna sta ormeggiando, il proprietario — un uomo dai tratti stanchi e gli occhi cerchiati di nero — sta scaricando casse avvolte in tele intrise di cera. È il momento di decidere come muoverti in questo dedalo di ponti e ombre.",
  "choices": [
    {
      "id": 1,
      "text": "Avvicinarsi al battello e cercare lavoro come scaricatrice o corriere improvvisato.",
      "intent": "social",
      "related_stats": ["vig", "pre"]
    },
    {
      "id": 2,
      "text": "Osservare le casse dall'ombra, cercando di capire se contengano qualcosa di valore che valga la pena sottrarre.",
      "intent": "stealth",
      "related_stats": ["agi", "ing"]
    },
    {
      "id": 3,
      "text": "Allontanarsi verso il mercato interno per cercare informazioni su chi sta accumulando reliquie in questo periodo.",
      "intent": "explore",
      "related_stats": ["ing", "occ"]
    }
  ],
  "mood": "tense",
  "location": "Banchina delle Nebbie",
  "scene_type": "introduzione",
  "entities_mentioned": [

...[truncated]...
```

### Dialogue scene with renderer metadata

- Compat score: `90.0/100`
- Duration: `10.078 s`
- Completion tokens: `865`
- Prompt tokens: `4674`
- Estimated cost: `$0.002466`
- Throughput: `85.8 tok/s`, `269.6 char/s`
- Notes:
  - event_callouts assenti

```text
{
  "narrative": "Lyanna stringe gli occhi, il suo sguardo scivola oltre la tua spalla verso le torri di guardia che lampeggiano di una luce fredda e innaturale. Si scosta il cappuccio, lasciando che l'umidità le appiccichi i capelli sulla fronte. 'Le campane non suonano per annunciare il tempo, Nerea. Hanno iniziato a mormorare nomi. Il mio, il tuo... e quello di chiunque abbia toccato quella maledetta reliquia.' \n\nFa una pausa, battendo due volte le dita sul legno marcio del tavolo, un tic nervoso che conosco bene. 'Le pattuglie non cercano te, o almeno non ancora. Cercano l'eco che ti porti appresso. Se la reliquia non viene sigillata col sale prima che il prossimo rintocco colpisca l'acqua, non saremo solo noi a dover scappare, ma l'intero distretto.'",
  "choices": [
    {
      "id": 1,
      "text": "Chiedere a Lyanna di aiutarmi a sigillare la reliquia col sale",
      "intent": "social",
      "risk": "medium"
    },
    {
      "id": 2,
      "text": "Cercare di nascondere la reliquia in un sarcofago di sale di scarto",
      "intent": "stealth",
      "risk": "low"
    },
    {
      "id": 3,
      "text": "Ignorare il pericolo e cercare di vendere la reliquia al mercato nero",
      "intent": "explore",
      "risk": "high"
    }
  ],
  "mood": "dark",
  "location": "Old Harbor",
  "scene_type": "dialogue",
  "dialogue_blocks": [
    {
      "speaker": "Lyanna Voss",
      "role": "npc",
      "text": "Le campane non suonano per annunciare il tempo, Nerea. Hanno iniziato a mormorare nomi. Il mio, il tuo... e quello di chiunque abbia toccato quella maledetta reliquia."
    },
    {
      "speaker": "Lyanna Voss",
      "role": "npc",
      "text": "Le pattuglie non cercano te, o almeno non ancora. Cercano l'eco che ti porti appresso. Se la reliquia non vie
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `100.0/100`
- Duration: `8.550 s`
- Completion tokens: `491`
- Prompt tokens: `4441`
- Estimated cost: `$0.001847`
- Throughput: `57.4 tok/s`, `181.4 char/s`

```text
{
  "narrative": "Afferri la fune con una stretta disperata, lasciando che il momentum del tuo scatto ti proietti oltre il vuoto. Sotto di te, il canale è una gola di acqua nera e rottami che ruggisce come una bestia affamata. Mentre il ponte di vetro si sgretola in mille frammenti luminosi dietro le tue spalle, senti il sibilo del ferro dei lanternieri che taglia l'aria a pochi centimetri dal tuo fianco. Il tuo corpo oscilla pericolosamente sospeso sul baratro, le mani bruciano per l'attrito della corda ruvida, ma il tuo istinto da corriere ti tiene incollata al percorso. Sei a metà strada, e la sponda opposta sembra allontanarsi a ogni folata di vento salmastro.",
  "choices": [
    {
      "id": 1,
      "text": "Usa lo slancio per darti una spinta e atterrare sulla sponda opposta"
    },
    {
      "id": 2,
      "text": "Resta immobile e cerca di nasconderti nell'ombra del pilone di arrivo"
    },
    {
      "id": 3,
      "text": "Lancia un urlo per attirare l'attenzione di un battelliere di passaggio"
    }
  ],
  "mood": "tense",
  "location": "Stormglass Causeway",
  "scene_type": "travel",
  "challenges": [
    {
      "type": "dice_roll",
      "difficulty": 50,
      "description": "Mantieni l'equilibrio sulla fune mentre il ponte crolla",
      "modifiers": [
        {
          "source": "Agilità",
          "value": 20
        },
        {
          "source": "Corsa",
          "value": 10
        }
      ]
    }
  ],
  "state_changes": {
    "skill_xp": {
      "skill": "Corsa",
      "xp": 15
    }
  }
}
```

### Chapter title + summary

- Compat score: `80.0/100`
- Duration: `4.554 s`
- Completion tokens: `349`
- Prompt tokens: `575`
- Estimated cost: `$0.000667`
- Throughput: `76.6 tok/s`, `294.2 char/s`
- Notes:
  - Summary fuori target: 194 parole

```text
```json
{
  "title": "Echi del Coro del Sale",
  "summary": "Nerea giunge alla Banchina delle Nebbie, portando con sé una mappa rubata e il peso opprimente dei debiti del fratello. Dopo aver negoziato un passaggio clandestino con un battelliere sospettoso, che identifica il sigillo spezzato sul suo polso, Nerea si addentra nel Mercato della Marea. Qui incontra Lyanna Voss, un'informatrice che rivela la caccia serrata delle Guardie di Marea per una reliquia scomparsa. Nerea confida a Lyanna di possedere l'oggetto, scoprendo che appartiene al misterioso Coro del Sale. Condotta in un magazzino allagato, Nerea utilizza il suo coltello per forzare la reliquia, innescando una visione traumatica: suo fratello, inginocchiato davanti a una campana nera, braccato dalle Guardie. La scoperta attira l'attenzione delle autorità, costringendo Nerea e Lyanna a una fuga precipitosa attraverso i ponti pericolanti del mercato. Nonostante la perdita del suo sacchetto di sale durante il crollo di un ponte, Nerea viene salvata da Lyanna, che decide di unirsi alla sua missione verso l'Old Harbor. Il capitolo si conclude con il rintocco inquietante delle campane sommerse e la consapevolezza che il destino di Nerea è ora indissolubilmente legato ai segreti oscuri del Coro del Sale, aggiungendo un nuovo, pericoloso debito al suo nome."
}
```
```

## x-ai/grok-4.1-fast

- Context: `2000000`
- Max completion: `30000`
- Prompt cost/token: `$0.000000200`
- Completion cost/token: `$0.000000500`
- Avg latency: `0.723 s`
- Avg cost: `$0.001399`
- Total benchmark cost: `$0.006994`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `0.942 s`
- Completion tokens: `2482`
- Prompt tokens: `1106`
- Estimated cost: `$0.001462`
- Throughput: `2633.6 tok/s`, `3614.0 char/s`

```text
```json
{
  "name": "Vespera: Sussurri del Mare",
  "description": "Vespera è una città-laguna fantasy decadente ispirata a una Venezia corrotta, dove canali salmastri e isole nere ospitano intrighi politici, culti delle campane sommerse e corporazioni mercantili spietate. Esplora quartieri allagati pieni di reliquie, affronta guardie corrotte e nebbie senzienti in un tono malinconico ma avventuroso. La magia dal suono delle campane ha un costo elevato, e ogni eco può alterare destini e memorie.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico ma Avventuroso",
  "setting": {
    "world_name": "Vespera",
    "era": "Era della Decadenza Lagunare",
    "geography": "Città-laguna su isole nere collegate da canali di acqua salmastra, con palazzi affondanti, porti nebbiosi e quartieri perennemente allagati pieni di reliquie.",
    "magic_system": "La magia nasce dal suono delle campane sommerse; le casate nobili ne controllano gli echi, ma ha sempre un costo personale e può alterare memoria e volontà.",
    "technology_level": "Rinascimentale decadente con gondole magiche, balestre e armi da mischia; il sale è usato come protettivo contro i sussurri del mare.",
    "society": "Stratificata tra nobili di canale, corporazioni mercantili spietate, cultisti del mare, guardie di porto corrotte e scavatori di relitti; i giuramenti pubblici hanno valore legale e spirituale.",
    "rules": [
      "La magia ha sempre un costo personale (salute, memoria o volontà).",
      "Il sale protegge dai sussurri del mare e dalle influenze magiche.",
      "Le campane sommerse alterano memoria e volontà di chi le ascolta.",
      "I giuramenti pubblici hanno valore legale e spirituale.",
      "Il debito non pagato può essere venduto come servitù perpetua."
    ],
    "factions"
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `0.775 s`
- Completion tokens: `1404`
- Prompt tokens: `4269`
- Estimated cost: `$0.001556`
- Throughput: `1812.4 tok/s`, `2017.6 char/s`

```text
{
  "narrative": "La nebbia avvolge la Banchina delle Nebbie come un sudario umido, le banchine nere luccicanti di salmastro e alghe marce. L'aria è densa di salsedine e ruggine, interrotta dal cupo rintocco di campane sommerse che vibra nelle ossa, piegando i ricordi e la volontà come un'onda traditrice. Sei Nerea, ex corriere delle banchine orientali, cresciuta tra mappe rubate e debiti familiari che ora ti stringono come catene legali. Nel tuo borsello, 8 corone di sale tintinnano debolmente—appena sufficienti per un'informazione o un passaggio su una gondola. Vespera si estende davanti a te: ponti bassi su canali salmastri, isole collegate da cordami sfilacciati, dove casate nobili tramano, scavatori di relitti sfidano le maree spezzate e il Coro del Sale custodisce reliquie pericolose. Il tuo debito preme: 2 corone dovute entro la prossima marea alta, o rischi la servitù.",
  "choices": [
    {"id": 1, "text": "Esaminare la banchina in cerca di opportunità o pericoli"},
    {"id": 2, "text": "Avvicinarsi al gruppo di scavatori che litiga su un relitto emerso"},
    {"id": 3, "text": "Dirigerti verso il Ponte Orientale, dominio della Casata Valcerra"},
    {"id": 4, "text": "Controllare la tua mappa rubata per contatti o vie nascoste"}
  ],
  "mood": "mysterious",
  "location": "Banchina delle Nebbie",
  "scene_type": "introduction",
  "dialogue_blocks": [],
  "entities_mentioned": [],
  "event_callouts": [],
  "state_changes": {},
  "challenges": [],
  "achievement_earned": null,
  "chapter_end": false,
  "chapter_title": ""
}
```

### Dialogue scene with renderer metadata

- Compat score: `85.0/100`
- Duration: `0.528 s`
- Completion tokens: `1600`
- Prompt tokens: `4517`
- Estimated cost: `$0.001703`
- Throughput: `3031.3 tok/s`, `3954.0 char/s`
- Notes:
  - event_callouts assenti
  - semantic metadata scarso sulle choices

```text
```json
{
  "narrative": "Mantieni la voce bassa, un sussurro che si mescola al ticchettio della pioggia sui moli. Lyanna ti fissa, i suoi occhi stanchi che ti squadrano nel buio. Tocca due volte il bordo della tettoia di legno, un gesto rapido e quasi impercettibile, prima di rispondere con il suo tono asciutto e tagliente.\n\n\"Sotto le campane? Voci vecchie, Nerea. Giuramenti che il sale non ha sigillato. Ricordi che non sono tuoi, ma ti si appiccicano addosso come alghe.\" Si interrompe, lancia un'occhiata alle luci lontane delle pattuglie. \"Cercano reliquie. Qualsiasi cosa luccichi o sussurri. Non sanno di te... ancora. Ma se hai qualcosa, meglio sparire prima che le campane risuonino di nuovo.\"",
  "choices": [
    {"id": 1, "text": "Mostrale la reliquia per vedere se la riconosce"},
    {"id": 2, "text": "Chiedile un piano per uscire dal porto"},
    {"id": 3, "text": "Offrile una parte del tuo sale per il suo aiuto"},
    {"id": 4, "text": "Ringraziala e dirigiti verso i Passi del Campanile da sola"}
  ],
  "mood": "tense",
  "location": "Old Harbor",
  "scene_type": "dialogue",
  "dialogue_blocks": [
    {"speaker": "Lyanna Voss", "role": "npc", "text": "Sotto le campane? Voci vecchie, Nerea. Giuramenti che il sale non ha sigillato. Ricordi che non sono tuoi, ma ti si appiccicano addosso come alghe."},
    {"speaker": "Lyanna Voss", "role": "npc", "text": "Cercano reliquie. Qualsiasi cosa luccichi o sussurri. Non sanno di te... ancora. Ma se hai qualcosa, meglio sparire prima che le campane risuonino di nuovo."}
  ],
  "entities_mentioned": [
    {"name": "Lyanna Voss", "type": "npc"}
  ],
  "state_changes": {
    "npc_disposition": {"name": "Lyanna Voss", "change": 8},
    "npc_thoughts": {"name": "Lyanna Voss", "thought": "Nerea è diretta ma cauta; potrebb
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `100.0/100`
- Duration: `0.814 s`
- Completion tokens: `1553`
- Prompt tokens: `4293`
- Estimated cost: `$0.001635`
- Throughput: `1908.7 tok/s`, `1843.6 char/s`

```text
```json
{
  "narrative": "Schizzi verso la fune laterale, le dita che stringono la corda fradicia e scivolosa mentre il ponte di Stormglass geme, si crepa e inizia a crollare in un fragore di schegge di vetro e acqua spumeggiante. Le catene dei lanternieri fischiano sopra la tua testa, sfiorandoti i capelli, mentre il canale sotto ribolle di detriti. La fune oscilla violentemente, tirata dalle correnti e dai tuoi movimenti: un passo falso e precipiti nel vuoto. Devi bilanciare perfettamente per attraversarla intera.\\n\\nIl cuore ti martella, lo stress sale alle stelle.",
  "choices": [
    {"id": 1, "text": "Concentrati sull'equilibrio e corri senza voltarti indietro"},
    {"id": 2, "text": "Usa la tua conoscenza dei canali per prevedere gli oscilli della fune"},
    {"id": 3, "text": "Lancia indietro il sacchetto di sale per distrarli definitivamente"},
    {"id": 4, "text": "Tenta un salto verso la sponda opposta se la fune cede"}
  ],
  "mood": "desperate",
  "location": "Stormglass Causeway",
  "scene_type": "travel",
  "challenges": [
    {
      "type": "dice_roll",
      "difficulty": 65,
      "description": "Attraversare la fune oscillante tra crollo del ponte e catene dei lanternieri",
      "modifiers": [
        {"source": "Agilità (4)", "value": 20},
        {"source": "Skill Corsa (lv1)", "value": 10},
        {"source": "Trait Tenace", "value": 5}
      ]
    }
  ],
  "state_changes": {
    "vitals": {
      "stress": {"current": 5, "max": 5}
    }
  }
}
```
```

### Chapter title + summary

- Compat score: `85.0/100`
- Duration: `0.561 s`
- Completion tokens: `1004`
- Prompt tokens: `679`
- Estimated cost: `$0.000638`
- Throughput: `1789.4 tok/s`, `2555.8 char/s`
- Notes:
  - Copertura limitata di eventi/entità attese

```text
```json
{
  "title": "Relic's Submerged Echoes",
  "summary": "Nerea arrives at the Foggy Dock with a stolen map sewn into her coat and her brother's name etched in a debt ledger. She discreetly negotiates passage with the ferryman, who accepts half payment but notices the broken seal on her wrist, deducing she's fleeing pursuers. Following her lead, she reaches the Tide Market, where she encounters Lyanna Voss. Lyanna warns of Tide Guards hunting a missing bell relic that vanished at dawn. Nerea reveals her relic, and Lyanna identifies its Salt Choir seal, leading her to a flooded warehouse where a vibrating salt altar hums with submerged bell tolls. Touching the relic with her knife tip cracks it open, unveiling a memory fragment: Nerea's brother kneeling before a black bell as Guards pound at the door. Alarmed, Nerea decides to flee before capture. During their escape, a bridge collapses; Nerea loses her salt pouch to the chaos but Lyanna saves her from a sudden flood and pledges aid to reach the Old Harbor. The chapter closes on Nerea's vow to uncover her brother's ties to the Salt Choir, as submerged bells resume ringing and a new debt is carved in her name. Key developments include partial relic activation revealing familial secrets, forging an alliance with Lyanna, and escalating pursuit by Tide Guards. No new items or abilities gained, but losses heighten stakes in this misty, tide-haunted world."
}
```
```

## qwen/qwen3.5-flash-02-23

- Context: `1000000`
- Max completion: `65536`
- Prompt cost/token: `$0.000000065`
- Completion cost/token: `$0.000000260`
- Avg latency: `0.680 s`
- Avg cost: `$0.000803`
- Total benchmark cost: `$0.004015`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `0.484 s`
- Completion tokens: `4222`
- Prompt tokens: `1033`
- Estimated cost: `$0.001165`
- Throughput: `8725.5 tok/s`, `4571.5 char/s`

```text
{
  "name": "Vespera: L'Eco della Laguna",
  "description": "Un fantasy oscuro ispirato a una Venezia decadente, dove la politica si intreccia con culti del mare in un'atmosfera malinconica ma avventurosa.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico ma Avventuroso",
  "setting": {
    "world_name": "Vespera",
    "era": "Età delle Campane Sommersi",
    "geography": "Città-laguna costruita su isole nere e canali di sale",
    "magic_system": "Magia basata sul suono delle campane sommerse e l'eco",
    "technology_level": "Medievale con elementi magici",
    "society": "Gerarchia nobiliare controllata da corporazioni mercantili spietate",
    "rules": [
      "La magia ha sempre un costo",
      "Il sale protegge dai sussurri del mare",
      "Le campane sommerse alterano memoria e volontà",
      "I giuramenti pubblici hanno valore legale e spirituale",
      "Il debito non pagato può essere venduto come servitù"
    ],
    "factions": [
      "Corporazioni Mercantili",
      "Religione delle Campane Sommersa",
      "Guardie di Porto Corrotte",
      "Casata Valcerra"
    ],
    "cultures": [
      "Nobili di canale",
      "Scavatori di relitti",
      "Pellegrini delle campane"
    ],
    "dangers": [
      "Nebbie senzienti",
      "Reliquie infette",
      "Inquisitori del suono",
      "Allagamenti improvvisi"
    ]
  },
  "stats_schema": {
    "vitals": [
      {"key": "hp", "label": "Salute", "starting": 10},
      {"key": "mana", "label": "Risonanza", "starting": 5},
      {"key": "stamina", "label": "Resistenza", "starting": 8}
    ],
    "attributes": [
      {"key": "str", "label": "Vigore", "starting": 3},
      {"key": "dex", "label": "Agilità", "starting": 3},
      {"key": "int", "label": "Ingegno", "starting": 3},
      {"key": "cha", "l
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `0.746 s`
- Completion tokens: `1145`
- Prompt tokens: `4391`
- Estimated cost: `$0.000583`
- Throughput: `1534.5 tok/s`, `3140.0 char/s`

```text
```json
{
  "narrative": "La nebbia ti avvolge mentre sali sulla Banchina delle Nebbie, il tuo respiro visibile nell'aria umida di Vespera. Le campane sommerse nell'acqua salmastra risuotano, un rintocco lontano che ti fa dimenticare per un istante dove stai andando. Il tuo nome è Nerea, ex corriere delle banchine, e hai due debiti di famiglia che ti inseguono come i predoni dei canali.\n\nIl legno scuro della banchina è scosso sotto il tuo peso, e sopra di te archi di pietra neri si perdono nella foschia. Due figure ti aspettano: una è l'ombra di un uomo con la divisa logora delle Guardie di Marea, l'altra, più vicina alla tua stazza, è un mercante con la Casata Valcerra stampata sul bordo del cappotto. La tua mano si stringe sul pugnale che porti nascosto sotto il mantello.\n\nIl rumore delle campane si fa più forte, e per un attimo senti il peso del tempo che si piega. La tua memoria vacilla, ma non ancora abbastanza da farti dimenticare il motivo per cui hai accettato questo incarico.",
  "choices": [
    {
      "id": 1,
      "text": "Interrogare la Guardia di Marea sui tuoi debiti",
      "intent": "social",
      "risk": "medium",
      "related_stats": ["pre", "occ"]
    },
    {
      "id": 2,
      "text": "Accostarsi al mercante Valcerra per informazioni",
      "intent": "social",
      "risk": "medium",
      "related_stats": ["pre", "ing"]
    },
    {
      "id": 3,
      "text": "Tentare di scivolare via attraverso i canali",
      "intent": "stealth",
      "risk": "high",
      "related_stats": ["agi", "occ"],
      "scope": "self"
    },
    {
      "id": 4,
      "text": "Osservare la scena con cautela prima di agire",
      "intent": "observe",
      "risk": "low",
      "related_stats": ["occ"]
    }
  ],
  "mood": "tense",
  "location": "Ba
...[truncated]...
```

### Dialogue scene with renderer metadata

- Compat score: `85.0/100`
- Duration: `0.837 s`
- Completion tokens: `1026`
- Prompt tokens: `4661`
- Estimated cost: `$0.000570`
- Throughput: `1225.4 tok/s`, `2755.3 char/s`
- Notes:
  - event_callouts assenti
  - semantic metadata scarso sulle choices

```text
```json
{
  "narrative": "Lyanna abbassa la voce quasi a un sussurro, gli occhi che si spostano verso il canale nero. \"Sotto le campane... non sentii le mie stesse membra. Sentii VOCE. Qualcuno che sapeva il mio nome prima che lo dicessi.\" \n\nUna pausa. Toca due volte il tavolo, come sempre quando mente o nasconde qualcosa. \"Le pattuglie non cercano te, Nerea. Cercano il suono. La reliquia che hai rubato canta ancora. E chi la sente, la trova.\" \n\nIl suo sguardo si allarga verso il passaggio secondario. \"Ma tu... hai un'idea di come stai cantando?\"",
  "choices": [
    {
      "id": 1,
      "text": "Chiedile se può aiutarla a nascondere il suono della reliquia"
    },
    {
      "id": 2,
      "text": "Mostrale la reliquia avvolta nel panno per vedere se la riconosce"
    },
    {
      "id": 3,
      "text": "Le chiedi se ha sentito altri suoni simili prima"
    },
    {
      "id": 4,
      "text": "Cambiare discorso e chiederle del pagamento"
    }
  ],
  "mood": "tense",
  "location": "Old Harbor",
  "scene_type": "dialogue",
  "dialogue_blocks": [
    {
      "speaker": "Lyanna Voss",
      "role": "npc",
      "text": "Sotto le campane... non sentii le mie stesse membra. Sentii VOCE. Qualcuno che sapeva il mio nome prima che lo dicessi."
    },
    {
      "speaker": "Lyanna Voss",
      "role": "npc",
      "text": "Le pattuglie non cercano te, Nerea. Cercano il suono. La reliquia che hai rubato canta ancora. E chi la sente, la trova."
    },
    {
      "speaker": "Lyanna Voss",
      "role": "npc",
      "text": "Ma tu... hai un'idea di come stai cantando?"
    }
  ],
  "entities_mentioned": [
    {
      "name": "Lyanna Voss",
      "type": "npc"
    },
    {
      "name": "Old Harbor",
      "type": "location"
    }
  ],
  "event_callouts": [],
  "
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `100.0/100`
- Duration: `0.790 s`
- Completion tokens: `845`
- Prompt tokens: `4428`
- Estimated cost: `$0.000508`
- Throughput: `1069.2 tok/s`, `2034.6 char/s`

```text
```json
{
  "narrative": "I tuoi piedi scivolano sul vetro bagnato. La fune laterale si tende con un urlo di metallo. Una delle catene dei lanternieri ti sfiora la spalla, lasciando scintille nere che bruciano come ghiaccio.\n\nIl ponte sotto di te cede con uno schianto. L'acqua nera ti inghiotte fino al ginocchio. Devi decidere: arrampicarti o restare in piedi.\n\nLe tue dita strizzano la fune. L'acqua sale. I lanternieri ridono, ma il loro ritmo si interrompe quando qualcosa si scontra contro i piloni.",
  "choices": [
    {"id": 1, "text": "Strappa via un pezzo di corda e lancialo contro i lanternieri"},
    {"id": 2, "text": "Usa la fune per lanciarti sul lato opposto"},
    {"id": 3, "text": "Annega sotto l'acqua e nuota verso i detriti"}
  ],
  "mood": "desperate",
  "location": "Stormglass Causeway",
  "scene_type": "danger",
  "dialogue_blocks": [],
  "entities_mentioned": [
    {"name": "Nerea", "type": "protagonist"},
    {"name": "Lanternieri", "type": "npc"}
  ],
  "event_callouts": [
    {"kind": "danger", "title": "Ponte in collasso", "detail": "Il ponte di vetro sta crollando nel canale"}
  ],
  "state_changes": {
    "vitals": {
      "hp": {
        "current": 7,
        "max": 10
      }
    }
  },
  "challenges": [
    {
      "type": "dice_roll",
      "difficulty": 55,
      "description": "Mantieni l'equilibrio mentre il ponte crolla",
      "modifiers": [
        {"source": "Skill Corsa (livello 1)", "value": 10},
        {"source": "Trait Tenace", "value": 5}
      ]
    }
  ],
  "achievement_earned": null,
  "chapter_end": false,
  "chapter_title": ""
}
```
```

### Chapter title + summary

- Compat score: `85.0/100`
- Duration: `0.544 s`
- Completion tokens: `4432`
- Prompt tokens: `572`
- Estimated cost: `$0.001190`
- Throughput: `8153.8 tok/s`, `3129.4 char/s`
- Notes:
  - Copertura limitata di eventi/entità attese

```text
{
  "title": "The Salt Choir's Hidden Bell",
  "summary": "Nerea arrives at the Foggy Wharf carrying a stolen map sewn into her coat and a debt register bearing her brother's name. Seeking covert passage, she negotiates with a suspicious ferryman who accepts half payment but notices the broken seal on her wrist, realizing she is fleeing pursuers. Following the trail, Nerea reaches the Tide Market, where she encounters Lyanna Voss. Lyanna warns that the Tide Guards are actively hunting a missing bell relic stolen at dawn. Nerea reveals the artifact, prompting Lyanna to recognize the distinctive Seal of the Salt Choir. She leads Nerea to a flooded warehouse containing a vibrating salt altar beneath the water. Using the tip of her knife, Nerea touches the relic, triggering a fragmented memory. She witnesses her brother kneeling before a black bell as the Guards knock at his door, confirming his involvement with the organization. Realizing the immediate danger, they flee the location. During the chaotic escape, a bridge collapses, causing Nerea to lose a bag of salt, but Lyanna saves her from a sudden flood surge. Impressed by her resilience, Lyanna agrees to help Nerea reach the Old Harbor. The chapter concludes with Nerea vowing to uncover her brother's connection to the Salt Choir. As night falls, submerged bells ring ominously, and a new debt is engraved upon her name, deepening the mystery of her pursuit. The journey highlights the tension between the Tide Guards and the Salt Choir, establishing a new alliance and raising the stakes regarding the brother's fate. The atmosphere grows darker as the water rises, signaling that the city itself is waking up to the conflict."
}
```

## google/gemini-2.5-flash-lite

- Context: `1048576`
- Max completion: `65535`
- Prompt cost/token: `$0.000000100`
- Completion cost/token: `$0.000000400`
- Avg latency: `0.617 s`
- Avg cost: `$0.000592`
- Total benchmark cost: `$0.002959`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `0.745 s`
- Completion tokens: `1314`
- Prompt tokens: `999`
- Estimated cost: `$0.000625`
- Throughput: `1762.6 tok/s`, `6018.9 char/s`

```text
```json
{
  "name": "Vespera: Echi Sommersi",
  "description": "Una città-laguna oscura e decadente, dove le antiche campane sommerse governano la magia e la politica è un gioco di potere tra casate nobiliari e culti marini. Naviga tra canali nebbiosi e quartieri allagati, affrontando pericoli ancestrali e i sussurri del mare.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico ma Avventuroso",
  "setting": {
    "world_name": "Vespera",
    "era": "Epoca di Decadenza Lagunare",
    "geography": "Una vasta città-laguna costruita su isole nere, con innumerevoli canali di acqua salmastra e quartieri parzialmente o totalmente allagati.",
    "magic_system": "La magia deriva dal suono delle campane sommerse, che possono alterare memoria, volontà e persino la realtà circostante. Il suo utilizzo ha sempre un costo.",
    "technology_level": "Rinascimentale con elementi magici",
    "society": "Strutturata in casate nobiliari che competono per il potere, corporazioni mercantili spietate, una religione incentrata sulle campane sommerse e una classe lavoratrice che vive ai margini della società.",
    "rules": [
      "La magia ha sempre un costo, spesso in termini di salute, sanità mentale o favori.",
      "Il sale puro è un potente scudo contro le influenze maligne e i sussurri del mare.",
      "Le campane sommerse non solo producono magia, ma possono alterare memoria e volontà di chi le ascolta.",
      "I giuramenti pubblici, pronunciati in luoghi sacri, hanno valore legale e spirituale vincolante.",
      "Il debito non pagato può portare alla servitù, con il debito stesso che può essere venduto come una merce."
    ],
    "factions": [
      "Casata Valcerra: Una delle più antiche e potenti casate nobiliari, nota per la sua spietatezza negli affari e il co
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `0.616 s`
- Completion tokens: `616`
- Prompt tokens: `4409`
- Estimated cost: `$0.000687`
- Throughput: `1000.5 tok/s`, `3378.3 char/s`

```text
```json
{
  "narrative": "Il sale. È la prima cosa che senti, che vedi, che vivi a Vespera. Non è il sale pulito e brillante dei mercanti, ma quello acre e denso che si insinua nelle narici, che impregna i vestiti e che ricopre le banchine nere come una seconda pelle.\n\nTi trovi sulla Banchina delle Nebbie, un luogo che porta il suo nome con crudele precisione. La nebbia, densa e lattiginosa, si aggrappa alle strutture fatiscenti di pontili marci e a cumuli di relitti arrugginiti che emergono dalle acque torbide del canale. Il suono delle campane sommerse, un lamento malinconico che vibra nelle ossa, è un costante promemoria dei sussurri che la città custodisce e dei debiti che legano le vite.\n\nLe tue mani stringono la stoffa ruvida del tuo mantello, un gesto automatico per combattere il freddo umido che penetra fin nelle ossa. Il tuo passato di corriere ti ha insegnato a muoverti in questo labirinto di canali e ponti bassi, ma Vespera muta costantemente, come le maree che scandiscono il tempo. Un nuovo debito pesa sulla tua famiglia, un fardello che senti gravare sulle spalle ad ogni respiro salmastro. La Casata Valcerra non perdona i ritardi, e il Coro del Sale custodisce segreti che è meglio non disturbare.",
  "choices": [
    {
      "id": 1,
      "text": "Dirigiti verso il mercato dei relitti, sperando di trovare qualcosa di valore."
    },
    {
      "id": 2,
      "text": "Cerca un contatto nelle Guardie di Marea, forse possono offrire informazioni o un lavoro."
    },
    {
      "id": 3,
      "text": "Dirigiti verso il Tempio del Coro del Sale, i sussurri delle campane potrebbero rivelare qualcosa."
    },
    {
      "id": 4,
      "text": "Osserva i dintorni, cercando tracce di attività insolita o pericoli imminenti."
    }
  ],
  "mood": "malinc
...[truncated]...
```

### Dialogue scene with renderer metadata

- Compat score: `90.0/100`
- Duration: `0.594 s`
- Completion tokens: `678`
- Prompt tokens: `4672`
- Estimated cost: `$0.000738`
- Throughput: `1141.8 tok/s`, `3519.6 char/s`
- Notes:
  - event_callouts assenti

```text
```json
{
  "narrative": "Lyanna ti guarda da sopra l'orlo del suo mantello unto. \"Le campane?\" Sussurra, la voce roca come il raschiare delle alghe contro lo scafo. \"Hanno portato ricordi. Giuramenti infranti. E il suono di qualcosa che si sta risvegliando nelle profondità.\" Si avvicina, il suo sguardo penetrante. \"Le pattuglie non cercano te, Nerea. Cercano *quella* cosa che hai rubato. E se ti trovano con essa, sarai solo un altro debito da saldare nel sale.\"",
  "choices": [
    {
      "id": 1,
      "text": "Mostrale la reliquia avvolta nel panno.",
      "intent": "social",
      "risk": "medium",
      "related_stats": [
        "pre"
      ]
    },
    {
      "id": 2,
      "text": "Chiedile dove potresti nascondere la reliquia.",
      "intent": "social",
      "risk": "medium",
      "related_stats": [
        "ing",
        "pre"
      ]
    },
    {
      "id": 3,
      "text": "Offrile una parte del suo debito in cambio di aiuto.",
      "intent": "social",
      "risk": "high",
      "related_stats": [
        "pre",
        "currency"
      ]
    }
  ],
  "mood": "tense",
  "location": "Old Harbor",
  "scene_type": "dialogue",
  "dialogue_blocks": [
    {
      "speaker": "Nerea",
      "role": "protagonist",
      "text": "Cosa hai sentito sotto le campane, Lyanna? E le pattuglie cercano me o la reliquia?"
    },
    {
      "speaker": "Lyanna Voss",
      "role": "npc",
      "text": "Le campane? Hanno portato ricordi. Giuramenti infranti. E il suono di qualcosa che si sta risvegliando nelle profondità. Le pattuglie non cercano te, Nerea. Cercano *quella* cosa che hai rubato. E se ti trovano con essa, sarai solo un altro debito da saldare nel sale."
    }
  ],
  "entities_mentioned": [
    {
      "name": "Lyanna Voss",
      "type": "npc"
   
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `100.0/100`
- Duration: `0.596 s`
- Completion tokens: `629`
- Prompt tokens: `4439`
- Estimated cost: `$0.000695`
- Throughput: `1054.7 tok/s`, `3160.6 char/s`

```text
```json
{
  "narrative": "Ti lanci verso la fune laterale, il tuo corpo che si tende mentre il ponte di vetro tempestato dietro di te cede con un fragore assordante. Le catene delle lanterne dei lanternieri sibilano nell'aria umida, colpendo il vuoto dove un istante prima c'era il ponte. Ogni passo sulla fune è una preghiera contro il vento e l'acqua che ti sferzano il viso. L'adrenalina ti pompa nelle vene, la tua \"Tenacia\" ti spinge avanti mentre cerchi di raggiungere la sicurezza della sponda opposta prima che anche questa via ti venga negata.",
  "choices": [
    {
      "id": 1,
      "text": "Continua a correre, puntando all'altro capo della fune.",
      "intent": "survive",
      "risk": "high",
      "related_stats": [
        "agi"
      ]
    },
    {
      "id": 2,
      "text": "Cerca di afferrare una delle catene delle lanterne per darti una spinta.",
      "intent": "survive",
      "risk": "extreme",
      "related_stats": [
        "agi",
        "vig"
      ]
    },
    {
      "id": 3,
      "text": "Lasciati cadere nel canale sottostante, sperando di sopravvivere all'impatto.",
      "intent": "survive",
      "risk": "high",
      "related_stats": [
        "vig"
      ]
    }
  ],
  "mood": "teso",
  "location": "Stormglass Causeway",
  "scene_type": "escape",
  "dialogue_blocks": [],
  "entities_mentioned": [],
  "event_callouts": [],
  "state_changes": {
    "vitals": {
      "hp": {
        "current": 7,
        "max": 10
      },
      "stress": {
        "current": 5,
        "max": 5
      }
    },
    "skill_xp": {
      "skill": "Corsa",
      "xp": 25
    },
    "skill_xp": {
      "skill": "Canali",
      "xp": 25
    }
  },
  "challenges": [
    {
      "type": "stat_check",
      "stat": "agi",
      "difficulty": 7,
      "descripti
...[truncated]...
```

### Chapter title + summary

- Compat score: `65.0/100`
- Duration: `0.535 s`
- Completion tokens: `388`
- Prompt tokens: `575`
- Estimated cost: `$0.000213`
- Throughput: `725.3 tok/s`, `2796.6 char/s`
- Notes:
  - Summary fuori target: 170 parole
  - Copertura limitata di eventi/entità attese

```text
```json
{
  "title": "Whispers of the Salt Choir",
  "summary": "Nerea arrives at the Fog Docks with a stolen map and her brother's name in a debt ledger. She secures passage with a ferryman who notices her broken wrist seal, indicating she's on the run. Following a lead, Nerea heads to the Tide Market. There, she encounters Lyanna Voss, who warns her about the Tide Guards searching for a stolen bell relic. Nerea reveals the relic, and Lyanna recognizes a Salt Choir seal. Lyanna leads Nerea to a flooded warehouse containing a salt altar that resonates with submerged bell tolls. Nerea touches the relic with her knife, triggering a memory fragment: her brother kneeling before a black bell as Guards approach. They flee, but a bridge collapses, causing Nerea to lose her salt pouch. Lyanna saves Nerea from a flash flood and agrees to help her reach the Old Harbor. The chapter concludes with Nerea vowing to uncover her brother's connection to the Salt Choir, as submerged bells begin to chime and a new debt is added to her name.",
  "new_npcs": [
    "Ferryman",
    "Lyanna Voss"
  ],
  "locations_visited": [
    "Fog Docks",
    "Tide Market",
    "Flooded Warehouse",
    "Old Harbor"
  ],
  "items_gained": [
    "Bell Relic (opened)"
  ],
  "abilities_gained": [],
  "world_changes": [
    "Submerged bells begin to chime again."
  ],
  "revelations": [
    "Nerea's brother was involved with the Salt Choir.",
    "The Tide Guards are searching for a stolen bell relic."
  ]
}
```
```

