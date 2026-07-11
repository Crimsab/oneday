# OneDay Model Benchmark

- Generated: `2026-04-08T22:23:28+02:00`
- Base URL: `https://openrouter.ai/api/v1`
- Mode: `json_schema`
- Scoring: `compatibility only` from the command output. Narrative quality is reviewed separately by hand.
- Raw JSON artifact: `2026-04-08-222716-oneday-benchmark-json_schema.json`

## Leaderboard

| Model | Compat Score | Success Rate | Avg Seconds | Avg Cost | Context | Max Out | Avg Tok/s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `qwen/qwen3.5-flash-02-23` | 91.5 | 100% | 0.627 | $0.000751 | 1000000 | 65536 | 3834.8 |
| `google/gemini-3.1-flash-lite-preview` | 89.2 | 100% | 8.916 | $0.001652 | 1048576 | 65536 | 73.2 |
| `x-ai/grok-4.1-fast` | 75.0 | 100% | 0.905 | $0.001462 | 2000000 | 30000 | 1694.4 |
| `google/gemini-2.5-flash-lite` | 70.8 | 100% | 0.964 | $0.000521 | 1048576 | 65535 | 539.5 |

## qwen/qwen3.5-flash-02-23

- Context: `1000000`
- Max completion: `65536`
- Prompt cost/token: `$0.000000065`
- Completion cost/token: `$0.000000260`
- Avg latency: `0.627 s`
- Avg cost: `$0.000751`
- Total benchmark cost: `$0.003755`

### Story creation final JSON

- Compat score: `85.0/100`
- Duration: `0.510 s`
- Completion tokens: `3986`
- Prompt tokens: `1035`
- Estimated cost: `$0.001104`
- Throughput: `7813.8 tok/s`, `4416.6 char/s`
- Notes:
  - Liste fuori target: rules=5 factions=5 cultures=3 dangers=4

```text
{
  "name": "Vespera",
  "description": "Una città-laguna costruita su isole nere e canali di sale, dove la magia nasce dal suono delle campane sommerse e le casate nobili cercano di controllarne gli echi. Un mondo di intrighi politici, culti marini e quartieri allagati, con un tono malinconico ma avventuroso.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico, Avventuroso, Cupo",
  "setting": {
    "world_name": "Vespera",
    "era": "Età delle Campane Sommersi",
    "geography": "Città-laguna su isole nere e canali di sale",
    "magic_system": "Magia del suono delle campane sommerse",
    "technology_level": "Medievale con elementi magici",
    "society": "Nobili casate mercantili e credenti del mare",
    "rules": [
      "La magia ha sempre un costo",
      "Il sale protegge dai sussurri del mare",
      "Le campane sommerse alterano memoria e volontà",
      "I giuramenti pubblici hanno valore legale e spirituale",
      "Il debito non pagato può essere venduto come servitù"
    ],
    "factions": [
      "Casata Valcerra",
      "Coro del Sale",
      "Corporazioni Mercantili",
      "Guardie di Marea",
      "Inquisitori del Suono"
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
      {"key": "hp", "label": "Vigore", "starting": 10},
      {"key": "mana", "label": "Eco", "starting": 5},
      {"key": "stamina", "label": "Resistenza", "starting": 8}
    ],
    "attributes": [
      {"key": "str", "label": "Forza", "starting": 3},
      {"key": "dex", "label": "Agilità", "starting": 3},
      {"key": "int", 
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `0.724 s`
- Completion tokens: `1025`
- Prompt tokens: `4393`
- Estimated cost: `$0.000552`
- Throughput: `1415.8 tok/s`, `2898.0 char/s`

```text
{
  "narrative": "La nebbia di Vespera non scende dalle nuvole — sale direttamente dal canale. \n\nTi stringi contro la banchina di pietra nera, le tue scarpe bagnate che scricchiolano sul selgio. Davanti a te, i palazzi della Casata Valcerra si ergono come tombe di marmo, le loro finestre oscurate che osservano in silenzio gli scavatori che cercano reliquie sommerse. Le campane dei corpi religiosi risuonano da qualche profondità acquatica, un suono che non dovrebbe esistere ma che ti fa tremare le ossa.\n\nUn mercante di Sale ti ha appena accennato a un lavoro — qualcuno ha visto una mappa rubata che potrebbe contenere le coordinate di un relitto prezioso nei canali settentrionali. Ma i debiti della tua famiglia continuano a crescere come le alghe, e le Guardie di Marea non hanno mai mostrato pietà per i corrieri in difficoltà.\n\nL'odore di sale puro e di acqua stagnante ti riempie i polmoni. Le campane sussurrano. E tu sai che il suono ha sempre un prezzo.",
  "choices": [
    {
      "id": 1,
      "text": "Cerca il mercante di Sale per più informazioni sul lavoro",
      "intent": "social",
      "risk": "medium",
      "related_stats": ["pre"]
    },
    {
      "id": 2,
      "text": "Esplora la banchina per trovare indizi sul relitto",
      "intent": "explore",
      "risk": "medium",
      "related_stats": ["occ"]
    },
    {
      "id": 3,
      "text": "Contatta un vecchio contatto tra gli scavatori di relitti",
      "intent": "social",
      "risk": "high",
      "related_stats": ["ing"]
    },
    {
      "id": 4,
      "text": "Rifletti sul prezzo da pagare — la mappa potrebbe costarti più di quanto pensi",
      "intent": "observe",
      "risk": "low",
      "related_stats": ["vol"]
    }
  ],
  "mood": "mysterious",
  "location": "Banchina
...[truncated]...
```

### Dialogue scene with renderer metadata

- Compat score: `75.0/100`
- Duration: `0.646 s`
- Completion tokens: `728`
- Prompt tokens: `4663`
- Estimated cost: `$0.000492`
- Throughput: `1126.9 tok/s`, `3232.2 char/s`
- Notes:
  - event_callouts assenti
  - semantic metadata scarso sulle choices

```text
{"narrnarrative":"La pioggia fina martella i pali del molo. Lyanna Voss ti osserva per un istante, le sue dita tolgono due volte il legno del molo prima di parlare. «Sotto le campane ho sentito qualcosa che non dovrebbero sentire i vivi», dice, il tono così basso da farti vibrare il petto. «Non sono le guardie che cercano te, Nerea. Cercano chi ha rubato la reliquia. Ma se tu hai la reliquia... le guardie non distinguono più.»\n\nI suoi occhi scuri si fissano sui tuoi. «Il Sale Choir sta facendo una caccia silenziosa. Se mi chiedi cosa ho sentito davvero, devo sapere se sei pronta ad ascoltare. Alcune cose non si cancellano una volta che le senti.»\n\nLa tensione nell'aria è palpabile. Le sirene delle pattuglie risuonano a distanza, il suono metallico che taglia la pioggia.","choices":[{"id":1,"text":"Dille di non preoccuparsi e chiedi cosa ha sentito davvero"},{"id":2,"text":"Mostrale la reliquia avvolta nel panno (se l'hai)"},{"id":3,"text":"Cambia argomento: chiedi come uscire dal porto senza essere vista"},{"id":4,"text":"Accetta il rischio e ascolta quello che ha da dire"}],"mood":"tense","location":"Old Harbor","scene_type":"dialogue","dialogue_blocks":[{"speaker":"Lyanna","role":"npc","text":"«Sotto le campane ho sentito qualcosa che non dovrebbero sentire i vivi.»"},{"speaker":"Lyanna","role":"npc","text":"«Non sono le guardie che cercano te, Nerea. Cercano chi ha rubato la reliquia.»"},{"speaker":"Lyanna","role":"npc","text":"«Il Sale Choir sta facendo una caccia silenziosa.»"}],"entities_mentioned":[{"name":"Lyanna Voss","type":"npc"},{"name":"Old Harbor","type":"location"},{"name":"Dock Wardens","type":"faction"},{"name":"Salt Choir","type":"faction"}],"state_changes":{"npc_disposition":{"name":"Lyanna Voss","change":10},"npc_thoughts":{"name":
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `100.0/100`
- Duration: `0.714 s`
- Completion tokens: `575`
- Prompt tokens: `4430`
- Estimated cost: `$0.000437`
- Throughput: `805.5 tok/s`, `1618.0 char/s`

```text
{"narrative":"Ti lanci verso la fune laterale mentre il ponte di vetro si frantuma sotto i tuoi piedi. I lanternieri urlano, le loro catene cariche di sale incandescente sferzate contro il muro d'acqua. La corrente ti trascina, una mano ti afferra la caviglia, e devi scegliere: saltare o trattenerti.\n\nUn granello di sale infuocato ti colpisce la guancia. Il calore brucia. La fune vibra come una corda di violino in un'orchestra infernale.","choices":[{"id":1,"text":"Salta sulla fune e tieniti stretto con entrambe le mani"},{"id":2,"text":"Taglia la catena con la lama nascosta nella manica"},{"id":3,"text":"Lascia la fune e tuffati nell'acqua in piena"}],"mood":"desperate","location":"Stormglass Causeway","scene_type":"combat","challenges":[{"type":"dice_roll","difficulty":70,"description":"Attento equilibrio sulla fune in frantumi","modifiers":[{"source":"Skill Corsa (livello 1)","value":10},{"source":"Agilità","value":4}]}],"entities_mentioned":[{"name":"Lanternieri","type":"enemy"},{"name":"Fune laterale","type":"object"}],"event_callouts":[{"kind":"danger","title":"Ponte in frantumi","detail":"Il passaggio principale è crollato"}]}
```

### Chapter title + summary

- Compat score: `100.0/100`
- Duration: `0.544 s`
- Completion tokens: `4355`
- Prompt tokens: `574`
- Estimated cost: `$0.001170`
- Throughput: `8011.8 tok/s`, `3592.9 char/s`

```text
{
  "title": "Campane Nere e Debito Sommerso",
  "summary": "Nerea fa il suo ingresso nella Banchina delle Nebbie, portando con sé una mappa rubata cucita nel cappotto e il nome del fratello ancora segnato in un registro di debiti. In un primo momento, cerca di passare inosservata parlando con il battelliere, ma quest'ultimo accetta solo metà del pagamento richiesto, notando immediatamente il sigillo spezzato sul suo polso. Questa scoperta rivela che la protagonista è in fuga da qualcuno, aumentando la tensione della scena e ponendo le basi per il conflitto futuro. Successivamente, Nerea segue una pista verso il Mercato della Marea, dove incontra Lyanna Voss. Lyanna avverte Nerea che le Guardie di Marea stanno cercando disperatamente una reliquia campanaria scomparsa all'alba, creando un senso di urgenza. Nerea mostra la reliquia a Lyanna, che riconosce immediatamente il sigillo del Coro del Sale. La donna la guida in un magazzino allagato, dove un vecchio altare di sale vibra a ogni rintocco sommerso, rivelando un'atmosfera magica e pericolosa. Con cautela, Nerea tocca la reliquia con la punta del coltello, provocando un'apertura parziale che rivela un frammento di memoria: una visione di suo fratello inginocchiato davanti a una campana nera, mentre le Guardie bussano alla porta. La rivelazione scatena una fuga precipitosa. Durante la corsa, un ponte cede sotto i loro piedi, causando la perdita del sacchetto di sale, un oggetto importante per il viaggio. Tuttavia, Lyanna salva Nerea da una piena improvvisa e decide di unirsi a lei per raggiungere l'Old Harbor, formando un'alleanza temporanea. Il capitolo si conclude con Nerea che promette di scoprire il motivo per cui suo fratello era legato al Coro del Sale. La notte cala mentre le campane sommerse tornano a suonar
...[truncated]...
```

## google/gemini-3.1-flash-lite-preview

- Context: `1048576`
- Max completion: `65536`
- Prompt cost/token: `$0.000000250`
- Completion cost/token: `$0.000001500`
- Avg latency: `8.916 s`
- Avg cost: `$0.001652`
- Total benchmark cost: `$0.008260`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `10.034 s`
- Completion tokens: `885`
- Prompt tokens: `1903`
- Estimated cost: `$0.001803`
- Throughput: `88.2 tok/s`, `273.5 char/s`

```text
{
  "name": "Vespera: L'Eco delle Campane",
  "description": "Un fantasy oscuro ambientato in una Venezia decadente e sommersa, dove la politica spietata si intreccia con culti marini proibiti. I giocatori navigheranno tra canali di sale e rovine allagate in cerca di potere e reliquie perdute.",
  "genre": "Dark Fantasy",
  "tone": "Malinconico, intrigante e avventuroso",
  "setting": {
    "world_name": "Vespera",
    "era": "Era del Declino Salino",
    "geography": "Un labirinto di isole nere e canali perennemente nebbiosi, con quartieri sommersi pieni di reliquie.",
    "magic_system": "Magia sonora basata sulle vibrazioni delle campane sommerse, capace di alterare memoria e realtà.",
    "technology_level": "Rinascimentale decadente",
    "society": "Strutturata in corporazioni mercantili spietate, guardie di porto corrotte e ordini religiosi devoti al mare.",
    "rules": [
      "La magia ha sempre un costo fisico o mentale",
      "Il sale protegge dai sussurri del mare",
      "Le campane sommerse alterano memoria e volontà",
      "I giuramenti pubblici hanno valore legale e spirituale",
      "Il debito non pagato può essere venduto come servitù"
    ],
    "factions": [
      "Casata Valcerra",
      "Coro del Sale",
      "Guardie di Marea"
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
      {
        "key": "hp",
        "label": "Salute",
        "starting": 20
      },
      {
        "key": "stamina",
        "label": "Resistenza",
        "starting": 15
      },
      {
        "key": "sanity",
     
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `9.761 s`
- Completion tokens: `566`
- Prompt tokens: `4411`
- Estimated cost: `$0.001952`
- Throughput: `58.0 tok/s`, `180.6 char/s`

```text
{
  "chapter_end": false,
  "choices": [
    {
      "id": 1,
      "intent": "explore",
      "risk": "low",
      "text": "Avvicinati al bordo della banchina per scrutare tra la nebbia."
    },
    {
      "id": 2,
      "intent": "social",
      "risk": "medium",
      "text": "Cerca qualcuno tra le ombre dei magazzini per chiedere notizie."
    },
    {
      "id": 3,
      "intent": "survive",
      "risk": "low",
      "text": "Controlla le tue scarse provviste e il peso del tuo debito."
    }
  ],
  "entities_mentioned": [
    {
      "name": "Banchina delle Nebbie",
      "type": "location"
    },
    {
      "name": "Vespera",
      "type": "location"
    }
  ],
  "event_callouts": [
    {
      "detail": "Ti trovi tra i canali salmastri di Vespera, dove il suono delle campane sommerse sembra sussurrare segreti dimenticati.",
      "kind": "setting",
      "title": "Benvenuta a Vespera"
    }
  ],
  "location": "Banchina delle Nebbie",
  "mood": "dark",
  "narrative": "Il freddo umido di Vespera ti entra nelle ossa come un ago di sale. Sei sulla Banchina delle Nebbie, un groviglio di legno marcito e catene arrugginite che pende pericolosamente sopra le acque nere del canale. Lontano, il rintocco sordo di una campana sommersa vibra contro la tua cassa toracica; non è un suono che si sente con le orecchie, ma una risonanza che scuote i ricordi e la volontà. Le luci delle lanterne a olio tremolano, soffocate dal vapore che sale dall'acqua salmastra. Hai le tasche quasi vuote e il peso del debito con la Casata Valcerra che ti preme sulla schiena come una pietra da macina. Qui, tra i ponti bassi e le banchine nere, la tua vita di corriere è appesa a un filo tanto sottile quanto la tua reputazione.",
  "scene_type": "arrival"
}
```

### Dialogue scene with renderer metadata

- Compat score: `90.0/100`
- Duration: `4.661 s`
- Completion tokens: `721`
- Prompt tokens: `4674`
- Estimated cost: `$0.002250`
- Throughput: `154.7 tok/s`, `483.3 char/s`
- Notes:
  - event_callouts assenti

```text
{
  "chapter_end": false,
  "choices": [
    {
      "id": 1,
      "intent": "social",
      "related_stats": [
        "pre",
        "ing"
      ],
      "risk": "medium",
      "text": "Mostrale la reliquia per convincerla a parlare"
    },
    {
      "id": 2,
      "intent": "stealth",
      "related_stats": [
        "agi",
        "ing"
      ],
      "risk": "high",
      "text": "Ignora le sue risposte e cerca un vicolo cieco per scappare"
    },
    {
      "id": 3,
      "intent": "lore",
      "related_stats": [
        "vol"
      ],
      "risk": "low",
      "text": "Insisti sulla natura del suono delle campane"
    }
  ],
  "dialogue_blocks": [
    {
      "role": "player",
      "speaker": "Nerea",
      "text": "Lyanna, cosa hai sentito veramente sotto le campane? Le pattuglie cercano me o questo pezzo di ferraglia?"
    },
    {
      "role": "npc",
      "speaker": "Lyanna Voss",
      "text": "Il rumore... non era metallo, Nerea. Era un battito. Qualcosa si è svegliato nel fango. Le guardie non cercano te, cercano il 'silenzio' che quella reliquia sta infrangendo. Se la tieni ancora, sei un faro nella nebbia."
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
  "location": "Old Harbor",
  "mood": "tense",
  "narrative": "Lyanna si sporge in avanti, il viso seminascosto dal cappuccio cerato. Le sue dita tamburellano nervose sul tavolo di legno marcio, un segno inequivocabile del suo disagio. Le sue parole sono un sussurro graffiante che si perde nel ritmo incessante della pioggia. Ti guarda negli occhi, e per un istante, il suo sguardo stanco sembra riflettere un terrore che va oltre le semplici guardie cittadine. La reliquia nella
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `85.0/100`
- Duration: `10.074 s`
- Completion tokens: `408`
- Prompt tokens: `4441`
- Estimated cost: `$0.001722`
- Throughput: `40.5 tok/s`, `134.0 char/s`
- Notes:
  - challenge presente ma con shape debole o incompleta

```text
{
  "challenges": [
    {
      "type": "dice_roll"
      },
      {
        "difficulty": 55,
        "skill": "Corsa",
        "stat": "agi",
        "type": "stat_check"
      }
    ],
    "choices": [
      {
        "id": 1,
        "text": "Continua a correre lungo la fune, ignorando il crollo"
      },
      {
        "id": 2,
        "text": "Salta verso un detrito galleggiante per ammortizzare la caduta"
      }
    ],
    "location": "Stormglass Causeway",
    "mood": "tense",
    "narrative": "Afferri la fune mentre il legname sotto i tuoi piedi si sbriciola come polvere di biscotto. Il fragore è assordante: i lanternieri urlano ordini, le loro aste colpiscono il vuoto dove un istante prima ti trovavi. Il ponte sta collassando in un vortice di detriti e acqua gelida. La fune oscilla violentemente sopra il canale in piena, e ogni tuo movimento rischia di scagliarti tra le onde scure sottostanti. Devi mantenere la concentrazione o il peso della reliquia che porti con te ti trascinerà a fondo.",
    "scene_type": "combat",
    "state_changes": {
      "challenges": [
        {
          "type": "dice_roll",
          "difficulty": 55,
          "description": "Mantieni l'equilibrio sulla fune mentre il ponte crolla",
          "modifiers": [
            "agi",
            "Corsa"
          ]
        }
      ]
    }
  }
```

### Chapter title + summary

- Compat score: `65.0/100`
- Duration: `10.050 s`
- Completion tokens: `249`
- Prompt tokens: `638`
- Estimated cost: `$0.000533`
- Throughput: `24.8 tok/s`, `118.1 char/s`
- Notes:
  - Summary fuori target: 196 parole
  - Copertura limitata di eventi/entità attese

```text
{
  "title": "Echoes of the Salt Choir",
  "summary": "Nerea arrives at the Foggy Quay, carrying a stolen map and a desperate need to find her brother, whose name is linked to a heavy debt. After securing passage from a suspicious boatman who recognizes her mark of flight, she travels to the Tide Market. There, she encounters Lyanna Voss, an informant who reveals that the Tide Guards are hunting for a missing bell relic. Nerea reveals that she possesses the very object they seek. Lyanna identifies the relic as belonging to the Salt Choir and leads Nerea to a flooded warehouse containing a vibrating salt altar. By touching the relic with her knife, Nerea unlocks a fragmented memory showing her brother kneeling before a black bell while guards close in on him. The discovery is cut short by a narrow escape; as they flee, a bridge collapses, forcing Nerea to sacrifice her pouch of salt. Lyanna saves her from the rising waters and pledges to assist her in reaching the Old Harbor. The chapter concludes with the haunting sound of submerged bells ringing out, signaling that Nerea is now inextricably bound to the mystery of the Salt Choir and burdened by a new, ominous debt."
}
```

## x-ai/grok-4.1-fast

- Context: `2000000`
- Max completion: `30000`
- Prompt cost/token: `$0.000000200`
- Completion cost/token: `$0.000000500`
- Avg latency: `0.905 s`
- Avg cost: `$0.001462`
- Total benchmark cost: `$0.007310`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `1.640 s`
- Completion tokens: `2934`
- Prompt tokens: `1539`
- Estimated cost: `$0.001775`
- Throughput: `1788.5 tok/s`, `1564.8 char/s`

```text
 {"description":"In Vespera, una città-laguna decadente ispirata a una Venezia sommersa da canali salmastri e nebbie eteree, nobili intriganti, mercanti spietati e cultisti del mare combattono per il controllo delle campane sommerse che donano potere magico a caro prezzo. Quartieri allagati brulicano di reliquie maledette, mentre politica corrotta e sussurri oceanici tessono un velo di malinconia su avventure audaci. Esplora un mondo oscuro dove ogni eco nasconde un segreto e ogni giuramento lega l'anima.","genre":"Fantasy Oscuro","name":"Vespera: Echi Sommersi","setting":{"cultures":["Nobili di Canale","Scavatori di Relitti","Pellegrini delle Campane"],"dangers":["Nebbie senzienti","Reliquie infette","Inquisitori del Suono","Allagamenti improvvisi"],"era":"Decadenza Rinascimentale","factions":["Casata Valcerra (nobili intriganti)","Coro del Sale (religione delle campane sommerse)","Guardie di Marea (corrotte)","Corporazione dei Mercanti Neri (spietati)"],"geography":"Città-laguna su isole nere basaltiche, canali di sale corrosivo, porti nebbiosi e quartieri perennemente allagati pieni di reliquie.","magic_system":"La magia nasce dal suono delle campane sommerse; controllarne gli echi altera memoria e volontà, richiede sale come catalizzatore e ha sempre un costo personale.","rules":["La magia ha sempre un costo.","Il sale protegge dai sussurri del mare.","Le campane sommerse alterano memoria e volontà.","I giuramenti pubblici hanno valore legale e spirituale.","Il debito non pagato può essere venduto come servitù."],"society":"Società stratificata con casate nobili di canale, corporazioni mercantili spietate, clero del Coro del Sale, guardie di porto corrotte e sottoproletariato degli scavatori di relitti.","technology_level":"Rinascimentale: gondole, balestre, 
...[truncated]...
```

### Narrative first turn

- Compat score: `90.0/100`
- Duration: `0.590 s`
- Completion tokens: `958`
- Prompt tokens: `4914`
- Estimated cost: `$0.001462`
- Throughput: `1623.8 tok/s`, `2278.1 char/s`

```text
 {"choices":[{"id":1,"text":"Dirigerti verso il Mercato del Sale, tra i banchi fumanti e i mercanti chiacchieroni","intent":"explore","risk":"low"},{"id":2,"text":"Cercare un contatto tra gli scavatori di relitti sulla banchina","intent":"social","risk":"medium"},{"id":3,"text":"Esaminare il canale vicino per mappe o relitti sommersi","intent":"investigation","risk":"medium","scope":"environment"}],"narrative":"L'aria salmastra di Vespera ti avvolge come un sudario umido, mentre poggi i piedi sulle assi viscide della Banchina delle Nebbie. È l'alba dell'Età delle Maree Spezzate, e la città lagunare si desta tra ponti bassi e banchine nere, un labirinto di canali che sussurrano segreti dimenticati. Le campane sommerse, reliquie di un'era perduta, echeggiano debolmente in lontananza, piegando la memoria come il sale corrode il ferro.\\n\\nSei Nerea, ex corriere delle banchine, cresciuta tra i debiti di famiglia e mappe rubate nei vicoli bui. Il tuo debito con la Casata Valcerra ammonta a 2 corone di sale, e i creditori non tarderanno. Hai 8 corone in tasca, la tua destrezza nei canali (livello 1) e un'ingegno affinato dalle strade. La nebbia senziente si infittisce, portando con sé il pericolo di allagamenti improvvisi o predoni nascosti.\\n\\nCosa fai?","mood":"mysterious","scene_type":"introduction","state_changes":{}}
```

### Dialogue scene with renderer metadata

- Compat score: `55.0/100`
- Duration: `0.722 s`
- Completion tokens: `1494`
- Prompt tokens: `5162`
- Estimated cost: `$0.001779`
- Throughput: `2069.0 tok/s`, `2606.3 char/s`
- Notes:
  - dialogue_blocks insufficienti o assenti
  - entities_mentioned non include Lyanna/Old Harbor
  - event_callouts assenti
  - semantic metadata scarso sulle choices

```text
 {"choices":[{"id":1,"text":"Mostrale la reliquia avvolta nel panno, per vedere se la riconosce"},{"id":2,"text":"Chiedile di contrabbandare la reliquia fuori dal porto"},{"id":3,"text":"Offri di ripagare un debito di famiglia in cambio di aiuto"},{"id":4,"text":"Sguscia via verso i vicoli allagati da sola"}],"narrative":"Mantieni la voce bassa, un sussurro che si mescola alla pioggia incessante. 'Cosa hai davvero sentito sotto le campane, Lyanna? E le pattuglie... cercano me o la reliquia?'\n\nLyanna ti fissa, gli occhi stanchi che ti squadrano. Si appoggia al palo umido, tamburellando due volte le dita contro il legno – un tic che conosci bene, ma stavolta non mente. 'Sotto le campane? Giuramenti spezzati, nomi di mercanti annegati da secoli. Risvegliano debiti che credevi pagati. Le pattuglie? Fermano chiunque puzzi di reliquie. La tua? Non sanno cos'è, ma i Dock Wardens la vogliono. Non te, ancora. Muoviti prima che cambino idea.'\n\nIl suo tono è tagliente come una lama, ma c'è un velo di rispetto nella sua occhiata: apprezza la tua cautela.","mood":"tense","scene_type":"dialogue","state_changes":{"npc_disposition":{"name":"Lyanna Voss","change":5},"npc_thoughts":{"name":"Lyanna Voss","thought":"Nerea è cauta, non impulsiva. Potrebbe valere la pena aiutarla un po' di più"},"npc_notes":{"name":"Lyanna Voss","note":"Ha tenuto la voce bassa e chiesto dettagli precisi senza esporsi"},"skill_xp":{"skill":"Furtività","xp":15},"entities_mentioned":[{"name":"Lyanna Voss","type":"npc"}],"dialogue_blocks":[{"speaker":"Lyanna Voss","role":"npc","text":"Sotto le campane? Giuramenti spezzati, nomi di mercanti annegati da secoli. Risvegliano debiti che credevi pagati. Le pattuglie? Fermano chiunque puzzi di reliquie. La tua? Non sanno cos'è, ma i Dock Wardens la voglion
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `55.0/100`
- Duration: `0.729 s`
- Completion tokens: `1347`
- Prompt tokens: `4938`
- Estimated cost: `$0.001661`
- Throughput: `1848.5 tok/s`, `1377.8 char/s`
- Notes:
  - challenges assente
  - challenge presente ma con shape debole o incompleta

```text
 {"choices":[{"id":1,"text":"Concentrati sul passaggio veloce"},{"id":2,"text":"Usa la tua conoscenza dei canali per anticipare l'onda"},{"id":3,"text":"Lancia indietro il sacchetto per rallentare gli inseguitori"}],"narrative":"Ti scatti verso la fune laterale, le dita che si aggrappano alle fibre ruvide impregnate di sale. Il ponte di Stormglass Causeway crolla con un fragore cristallino, schegge di vetro nero che schizzano ovunque mentre l'acqua ribolle furiosa sotto di te. Le catene dei lanternieri fischiano nell'aria, mancandoti di un soffio, e la fune si tende oscillando violentemente per la piena improvvisa. Il tuo cuore martella: un passo falso e sei cibo per i gorghi.\\n\\n[Devi superare questa prova per continuare]","mood":"desperate","scene_type":"chase","state_changes":{"skill_xp":{"Corsa":{"xp":25},"Canali":{"xp":15}},"world_location_add":"Sponda del Canale di Vetro Spezzato","world_event_add":"Il ponte di Stormglass Causeway è crollato, bloccando il passaggio centrale." }  }
```

### Chapter title + summary

- Compat score: `85.0/100`
- Duration: `0.845 s`
- Completion tokens: `965`
- Prompt tokens: `752`
- Estimated cost: `$0.000633`
- Throughput: `1142.1 tok/s`, `1750.4 char/s`
- Notes:
  - Copertura limitata di eventi/entità attese

```text
 {
  "summary": "Nerea arrives at the Foggy Dock with a stolen map sewn into her coat and her brother's name etched in a debt ledger. She discreetly negotiates passage with the ferryman, who accepts half payment but notices the broken seal on her wrist, deducing she's on the run. Following her lead, she reaches the Tide Market, where she encounters Lyanna Voss. Lyanna warns of Tide Guards hunting a missing dawn bell relic. Nerea reveals her relic, and Lyanna identifies its Salt Choir seal, leading her to a flooded warehouse with a vibrating salt altar attuned to submerged bell tolls. Probing the relic with her knife tip, Nerea unlocks a memory fragment: her brother kneeling before a black bell as Guards knock at the door. Alarmed, they flee as pursuit closes in. A collapsing bridge costs Nerea her salt pouch, but Lyanna saves her from a sudden flood and pledges aid to reach Old Harbor. The chapter ends with Nerea vowing to uncover her brother's ties to the Salt Choir, as submerged bells resume ringing and a new debt is carved in her name. Key decisions include stealthy bartering for passage, trusting Lyanna with the relic, and activating it despite risks. New NPCs: ferryman and Lyanna Voss, pivotal ally. Locations: Foggy Dock, Tide Market, flooded warehouse, en route to Old Harbor. Items lost: salt pouch; relic partially unlocked, revealing lore. Revelations hint at deeper Salt Choir intrigue and familial debts.",
  "title": "Relic's Whispered Betrayal"
}
```

## google/gemini-2.5-flash-lite

- Context: `1048576`
- Max completion: `65535`
- Prompt cost/token: `$0.000000100`
- Completion cost/token: `$0.000000400`
- Avg latency: `0.964 s`
- Avg cost: `$0.000521`
- Total benchmark cost: `$0.002603`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `1.713 s`
- Completion tokens: `1133`
- Prompt tokens: `999`
- Estimated cost: `$0.000553`
- Throughput: `661.5 tok/s`, `2222.8 char/s`

```text
{
  "name": "Echi di Vespera",
  "description": "Un'avventura fantasy oscura ambientata nella decadente città-laguna di Vespera, dove intrighi politici, culti marini e magia sonora plasmano un mondo malinconico e pericoloso.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico, Avventuroso, Cupo, Elegante",
  "setting": {
    "world_name": "Vespera",
    "era": "L'Era del Sale e del Silenzio",
    "geography": "Una vasta laguna salmastra punteggiata da isole nere artificiali, collegate da ponti fatiscenti e canali tortuosi. I quartieri più bassi sono perennemente allagati.",
    "magic_system": "La magia sonora, legata al suono delle campane sommerse che alterano memoria e volontà. La sua manipolazione richiede un sacrificio o un prezzo.",
    "technology_level": "Rinascimentale con elementi steampunk arcaici (es. ingranaggi mossi da maree, lanterne a olio salmastro).",
    "society": "Una rigida struttura di casate nobiliari mercantili, corporazioni spietate, una chiesa dedita ai culti del mare e una vasta popolazione di lavoratori e diseredati che lottano per sopravvivere tra le acque scure.",
    "rules": [
      "La magia ha sempre un costo, spesso personale o spirituale.",
      "Il sale puro protegge dai sussurri e dalle influenze del mare profondo.",
      "I giuramenti pronunciati pubblicamente hanno valore legale e spirituale, vincolando chi li pronuncia.",
      "Il debito non pagato può essere riscattato attraverso la servitù o la cessione di memorie.",
      "Le campane sommerse non sono solo fonti di magia, ma anche custodi di segreti antichi."
    ],
    "factions": [
      "Casata Valcerra: La più potente corporazione mercantile, nota per la sua spietatezza e i suoi traffici oscuri.",
      "Coro del Sale: La chiesa ufficiale, devota ai misteri del
...[truncated]...
```

### Narrative first turn

- Compat score: `80.0/100`
- Duration: `0.858 s`
- Completion tokens: `398`
- Prompt tokens: `4411`
- Estimated cost: `$0.000600`
- Throughput: `463.9 tok/s`, `1567.5 char/s`

```text
{
  "choices": [
    {
      "id": 1,
      "text": "Controlla le tue tasche per assicurarti che tutto sia al suo posto."
    },
    {
      "id": 2,
      "text": "Osserva l'ambiente circostante per individuare potenziali pericoli o alleati."
    },
    {
      "id": 3,
      "text": "Dirigiti verso la taverna più vicina per cercare informazioni o riparo."
    }
  ],
  "narrative": "La pioggia salmastra picchietta incessantemente sui bassi tetti di Vespera, ogni goccia un sussurro del mare infermo che circonda questa città lagunare.\nLe banchine nere, consumate dal sale e dal tempo, si estendono come dita scheletriche nel bruno torbido dei canali. L'aria è densa, pregna dell'odore di pesce marcio, alghe putrescenti e qualcosa di più antico, più sinistro, che affiora dalle profondità corrotte.\nSei Nerea, un tempo corriere delle banchine, ora solo un'altra anima persa nei dedali di ponti e calli di questa città malinconica. I debiti della tua famiglia ti perseguitano come ombre, un fardello più pesante del sudario di nebbia che avvolge le isole.\nLa tua reputazione è poco più che un sussurro, e le tue tasche contengono poche corone di sale, un magro conforto contro la marea crescente di problemi.",
  "state_changes": {
    "location": "Banchina delle Nebbie",
    "world_location_add": "Banchina delle Nebbie"
  }
}
```

### Dialogue scene with renderer metadata

- Compat score: `60.0/100`
- Duration: `0.885 s`
- Completion tokens: `445`
- Prompt tokens: `4672`
- Estimated cost: `$0.000645`
- Throughput: `502.6 tok/s`, `1569.8 char/s`
- Notes:
  - dialogue_blocks insufficienti o assenti
  - entities_mentioned non include Lyanna/Old Harbor
  - event_callouts assenti

```text
{
  "narrative": "Lyanna abbassa la voce, il suono quasi perso nel tambureggiare della pioggia. \"Sentito? Ho sentito il suono di vecchi rancori che riaffiorano. Come se il mare stesso stesse sussurrando promesse infrante. Le pattuglie... cercano chiunque abbia toccato qualcosa che non doveva. Ti hanno visto? Se sì, cercano te. Se no, cercano la reliquia. A seconda di chi incontri, una cosa o l'altra sarà peggio.\"",
  "choices": [
    {
      "id": 1,
      "text": "Mostrale la reliquia avvolta nel panno",
      "intent": "show_item",
      "risk": "medium",
      "scope": "npc"
    },
    {
      "id": 2,
      "text": "Chiedi se ha sentito anche il nome della reliquia",
      "intent": "lore",
      "risk": "low",
      "scope": "npc"
    },
    {
      "id": 3,
      "text": "Chiedi se c'è un modo per uscire dal porto senza essere visti",
      "intent": "explore",
      "risk": "medium",
      "scope": "environment"
    }
  ],
  "mood": "tense",
  "state_changes": {
    "npc_disposition": {
      "name": "Lyanna Voss",
      "change": 5
    },
    "npc_thoughts": {
      "name": "Lyanna Voss",
      "thought": "È più informata di quanto sembri. Spero non mi trascini nei suoi guai."
    },
    "npc_notes": {
      "name": "Lyanna Voss",
      "note": "Ha chiesto informazioni specifiche sul suono delle campane e sulle pattuglie, mostrando cautela."
    }
  }
}
```

### Challenge-producing action scene

- Compat score: `55.0/100`
- Duration: `0.825 s`
- Completion tokens: `523`
- Prompt tokens: `4439`
- Estimated cost: `$0.000653`
- Throughput: `633.7 tok/s`, `1956.7 char/s`
- Notes:
  - challenges assente
  - challenge presente ma con shape debole o incompleta

```text
{
  "narrative": "Ti lanci verso la fune, le mani che cercano disperatamente un appiglio mentre il ponte dietro di te cede con un boato assordante. Le catene delle lanterne, ora libere, sferzano l'aria umida sopra la tua testa. La fune è scivolosa, intrisa dell'acqua salmastra e del sudore della tua fuga, ma la tua agilità ti tiene in equilibrio precario. I lanternieri urlano insulti dalla sponda che stai abbandonando, ma sono troppo lontani per raggiungerti. Hai guadagnato terreno, ma la tua fuga ha lasciato una scia chiara.",
  "choices": [
    {
      "id": 1,
      "text": "Continua a correre verso la sponda opposta",
      "intent": "flee",
      "risk": "medium",
      "scope": "self"
    },
    {
      "id": 2,
      "text": "Cerca di fare qualche passo indietro sulla fune per guadagnare tempo",
      "intent": "survive",
      "risk": "high",
      "scope": "environment"
    },
    {
      "id": 3,
      "text": "Cerca di capire cosa hanno in mano i lanternieri",
      "intent": "observe",
      "risk": "low",
      "scope": "npc"
    }
  ],
  "mood": "teso",
  "state_changes": {
    "skill_xp": [
      {
        "skill": "Corsa",
        "xp": 25
      },
      {
        "skill": "Canali",
        "xp": 50
      }
    ],
    "vitals": {
      "hp": -1
    },
    "event_callouts": [
      {
        "title": "Ponte crollato",
        "detail": "Lo Stormglass Causeway è ora inagibile.",
        "kind": "location_change"
      },
      {
        "title": "Traccia lasciata",
        "detail": "I lanternieri ti hanno vista fuggire.",
        "kind": "consequence"
      }
    ]
  }
}
```

### Chapter title + summary

- Compat score: `65.0/100`
- Duration: `0.539 s`
- Completion tokens: `235`
- Prompt tokens: `573`
- Estimated cost: `$0.000151`
- Throughput: `436.0 tok/s`, `1925.7 char/s`
- Notes:
  - Summary fuori target: 171 parole
  - Copertura limitata di eventi/entità attese

```text
{
  "title": "Echoes of the Salt Choir",
  "summary": "Nerea arrives at the Misty Dock with a stolen map and her brother's name in a debt ledger. She secures passage with a wary ferryman who notices her broken wrist seal, indicating she's on the run. Following a lead, Nerea ventures into the Tide Market, where she encounters Lyanna Voss. Lyanna warns Nerea that the Tide Guards are searching for a missing bell relic. Nerea reveals the relic, and Lyanna recognizes a Salt Choir seal, leading Nerea to a flooded warehouse. There, an ancient salt altar hums with submerged chimes. Nerea touches the relic with her knife, triggering a fragmented memory: her brother kneeling before a black bell as guards approach. They flee as a bridge collapses, and Nerea loses her salt pouch, but Lyanna saves her from a flash flood and agrees to help her reach the Old Harbor. The chapter concludes with Nerea vowing to uncover her brother's connection to the Salt Choir, as submerged bells begin to chime again and a new debt is added to her name."
}
```

