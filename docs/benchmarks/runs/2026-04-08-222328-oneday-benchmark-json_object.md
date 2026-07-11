# OneDay Model Benchmark

- Generated: `2026-04-08T22:19:46+02:00`
- Base URL: `https://openrouter.ai/api/v1`
- Mode: `json_object`
- Scoring: `compatibility only` from the command output. Narrative quality is reviewed separately by hand.
- Raw JSON artifact: `2026-04-08-222328-oneday-benchmark-json_object.json`

## Leaderboard

| Model | Compat Score | Success Rate | Avg Seconds | Avg Cost | Context | Max Out | Avg Tok/s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `google/gemini-3.1-flash-lite-preview` | 92.2 | 100% | 5.371 | $0.001665 | 1048576 | 65536 | 302.7 |
| `qwen/qwen3.5-flash-02-23` | 91.0 | 100% | 0.582 | $0.000849 | 1000000 | 65536 | 4596.8 |
| `x-ai/grok-4.1-fast` | 90.2 | 100% | 0.550 | $0.001421 | 2000000 | 30000 | 2939.0 |
| `google/gemini-2.5-flash-lite` | 80.2 | 100% | 0.760 | $0.000519 | 1048576 | 65535 | 778.2 |

## google/gemini-3.1-flash-lite-preview

- Context: `1048576`
- Max completion: `65536`
- Prompt cost/token: `$0.000000250`
- Completion cost/token: `$0.000001500`
- Avg latency: `5.371 s`
- Avg cost: `$0.001665`
- Total benchmark cost: `$0.008325`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `0.825 s`
- Completion tokens: `786`
- Prompt tokens: `1005`
- Estimated cost: `$0.001430`
- Throughput: `953.2 tok/s`, `2925.1 char/s`

```text
{
  "name": "Vespera: Echi del Mare",
  "description": "Una Venezia decadente costruita su isole nere e canali di sale, dove il potere si misura in segreti e il suono delle campane sommerse altera la realtà. È un mondo di intrighi politici, reliquie maledette e una malinconia profonda che avvolge ogni canale.",
  "genre": "Dark Fantasy",
  "tone": "Malinconico, cupo, avventuroso",
  "setting": {
    "world_name": "Vespera",
    "era": "Era del Declino Salino",
    "geography": "Un labirinto di isole nere, canali nebbiosi e quartieri perennemente allagati.",
    "magic_system": "Magia basata sulla risonanza sonora delle campane sommerse; ogni incantesimo richiede un sacrificio di memoria o salute mentale.",
    "technology_level": "Rinascimentale decadente con elementi magici.",
    "society": "Una gerarchia rigida dominata da corporazioni mercantili spietate e una teocrazia basata sul culto del mare.",
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
    "vitals": [
      {"key": "hp", "label": "Salute", "starting": 20},
      {"key": "san", "label": "Sanità", "starting": 20},
      {"key": "sta", "label": "Stamina", "starting": 10}
    ],
    "a
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `8.802 s`
- Completion tokens: `652`
- Prompt tokens: `4411`
- Estimated cost: `$0.002081`
- Throughput: `74.1 tok/s`, `239.3 char/s`

```text
{
  "narrative": "La pioggia salmastra di Vespera batte ritmicamente contro i legni marci della Banchina delle Nebbie. L'aria è densa, satura del profumo di alghe putride e del rintocco lontano, ovattato, di una campana sommersa che sembra vibrare fin dentro le tue ossa. Sei Nerea, e le tue tasche sono leggere quanto il tuo animo: otto corone di sale e un debito che pende sul tuo capo come la lama di una ghigliottina. Davanti a te, le luci fioche delle lanterne ad olio della Casata Valcerra danzano sui canali neri, riflettendo un mondo che raramente offre una seconda possibilità a chi è nato tra i relitti.\n\nUn mercante della corporazione, avvolto in un mantello di lana grezza, sta caricando casse di vetro alchemico su una chiatta, ignorando deliberatamente la tua presenza. Poco lontano, una pattuglia delle Guardie di Marea sta ispezionando il carico di uno scavatore, le loro divise sporche di fango di canale e i volti segnati dal cinismo. Hai bisogno di monete, e in fretta.",
  "choices": [
    {
      "id": 1,
      "text": "Approcciare il mercante della Casata Valcerra per cercare lavoro come corriere.",
      "intent": "social",
      "risk": "low"
    },
    {
      "id": 2,
      "text": "Sfruttare la confusione del controllo doganale per rubare qualcosa dalla chiatta incustodita.",
      "intent": "stealth",
      "risk": "high"
    },
    {
      "id": 3,
      "text": "Ascoltare le conversazioni dei pescatori di relitti per scoprire voci su tesori emergenti.",
      "intent": "explore",
      "risk": "low"
    }
  ],
  "mood": "dark",
  "location": "Banchina delle Nebbie",
  "scene_type": "downtime",
  "entities_mentioned": [
    {"name": "Casata Valcerra", "type": "faction"},
    {"name": "Guardie di Marea", "type": "faction"}
  ],
  "state_changes": {
   
...[truncated]...
```

### Dialogue scene with renderer metadata

- Compat score: `90.0/100`
- Duration: `6.371 s`
- Completion tokens: `804`
- Prompt tokens: `4674`
- Estimated cost: `$0.002374`
- Throughput: `126.2 tok/s`, `410.6 char/s`
- Notes:
  - event_callouts assenti

```text
{
  "narrative": "Ti avvicini a Lyanna, evitando le pozzanghere che riflettono il bagliore fioco dei lampioni a gas. Abbassi la voce fin quasi a un soffio, mentre il rumore della pioggia copre le vostre parole. 'Cosa hai sentito sotto le campane, Lyanna? Sono qui per me o per quello che porto?'\n\nLei ti scruta, gli occhi stanchi che guizzano verso una pattuglia di Guardie del Molo che sta perquisendo un vecchio pescatore poco distante. Tocca nervosamente il bordo del tavolo di legno marcio due volte, un gesto che tradisce la sua inquietudine. 'Le campane non suonano per gli uomini, Nerea. Suonano quando il sale si incrina. E stamattina... hanno pianto un nome che somigliava maledettamente al tuo. Le pattuglie hanno ordini rigidi: non cercano contrabbandieri, cercano un 'eco'. Qualunque cosa tu abbia preso, sta attirando l'attenzione del Coro di Sale più di quanto tu possa immaginare.'",
  "choices": [
    {"id": 1, "text": "Mostrale la reliquia per capire cosa la rende così speciale", "intent": "observe", "risk": "medium", "related_stats": ["ing"]},
    {"id": 2, "text": "Chiedile di aiutarti a nascondere la traccia dell'eco", "intent": "social", "risk": "low", "related_stats": ["pre", "vol"]},
    {"id": 3, "text": "Ignora il pericolo e scappa verso i vicoli alti", "intent": "flee", "risk": "high", "related_stats": ["agi"]}
  ],
  "mood": "tense",
  "location": "Old Harbor",
  "scene_type": "dialogue",
  "dialogue_blocks": [
    {"speaker": "Nerea", "role": "protagonist", "text": "Cosa hai sentito sotto le campane, Lyanna? Sono qui per me o per quello che porto?"},
    {"speaker": "Lyanna Voss", "role": "npc", "text": "Le campane non suonano per gli uomini, Nerea. Suonano quando il sale si incrina. E stamattina... hanno pianto un nome che somigliava maledettamente a
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `100.0/100`
- Duration: `10.080 s`
- Completion tokens: `552`
- Prompt tokens: `4441`
- Estimated cost: `$0.001938`
- Throughput: `54.8 tok/s`, `175.2 char/s`

```text
{
  "narrative": "Il legno marcio cede con un gemito metallico proprio mentre balzi verso la fune di canapa. Il ponte precipita nel canale sottostante, trasformandosi in una pioggia di schegge e detriti che ingrossano la corrente. Resti sospesa nel vuoto, dondolando furiosamente sopra l'acqua vorticante, mentre le catene delle lanterne dei tuoi inseguitori frustano l'aria sopra la tua testa, mancando il bersaglio di pochi centimetri. Il riverbero del metallo contro la pietra riecheggia lungo tutto il passaggio. Hai il fiato corto, le nocche bianche per la presa serrata e il cuore che batte come un tamburo di guerra.",
  "choices": [
    {
      "id": 1,
      "text": "Continua a dondolare per raggiungere l'altra sponda con un balzo",
      "intent": "flee",
      "risk": "high",
      "related_stats": ["agi"]
    },
    {
      "id": 2,
      "text": "Resta immobile, cercando di mimetizzarti tra le ombre del pilone",
      "intent": "stealth",
      "risk": "medium",
      "related_stats": ["vol"]
    },
    {
      "id": 3,
      "text": "Urla una sfida ai lanternieri per attirarli verso la trappola del ponte crollato",
      "intent": "attack",
      "risk": "high",
      "related_stats": ["vig"]
    }
  ],
  "mood": "tense",
  "location": "Stormglass Causeway",
  "scene_type": "travel",
  "challenges": [
    {
      "type": "dice_roll",
      "difficulty": 55,
      "description": "Mantieni l'equilibrio sulla fune oscillante mentre i detriti colpiscono",
      "modifiers": [
        {
          "source": "Agilità",
          "value": 20
        },
        {
          "source": "Abilità Corsa",
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

- Compat score: `65.0/100`
- Duration: `0.779 s`
- Completion tokens: `238`
- Prompt tokens: `575`
- Estimated cost: `$0.000501`
- Throughput: `305.4 tok/s`, `1438.6 char/s`
- Notes:
  - Summary fuori target: 181 parole
  - Copertura limitata di eventi/entità attese

```text
{
  "title": "Echoes of the Salt Choir",
  "summary": "Nerea arrives at the Foggy Quay, carrying a stolen map and burdened by her brother's mounting debts. After securing passage from a suspicious boatman who identifies her as a fugitive, she navigates to the Tide Market. There, she encounters Lyanna Voss, who warns her that the Tide Guards are hunting for a missing bell-shaped relic. Nerea reveals the artifact to Lyanna, who identifies it as belonging to the mysterious Salt Choir. The pair retreats to a flooded warehouse containing an ancient salt altar, where Nerea uses her knife to force the relic open. This action triggers a haunting vision of her brother kneeling before a black bell while being hunted by the authorities. As they attempt to flee the area, a bridge collapses, forcing Nerea to abandon her supply of salt, though Lyanna manages to save her from a sudden surge of water. The chapter concludes with the two forming an alliance to reach Old Harbor, while the distant, submerged bells begin to toll once more, signaling that Nerea is now deeply entangled in the dark secrets of the Salt Choir."
}
```

## qwen/qwen3.5-flash-02-23

- Context: `1000000`
- Max completion: `65536`
- Prompt cost/token: `$0.000000065`
- Completion cost/token: `$0.000000260`
- Avg latency: `0.582 s`
- Avg cost: `$0.000849`
- Total benchmark cost: `$0.004246`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `0.584 s`
- Completion tokens: `5004`
- Prompt tokens: `1035`
- Estimated cost: `$0.001368`
- Throughput: `8565.9 tok/s`, `4431.9 char/s`

```text
{
  "name": "Vespera: L'Eco delle Campane",
  "description": "Una città-laguna decadente costruita su isole nere e canali di sale, dove la magia nasce dal suono delle campane sommerse e ogni casata nobile cerca di controllarne gli echi.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico ma Avventuroso",
  "setting": {
    "world_name": "Vespera",
    "era": "Era della Decadenza",
    "geography": "Isole nere e canali di sale, quartieri allagati pieni di reliquie",
    "magic_system": "Magia acustica delle campane sommerse",
    "technology_level": "Medievale con elementi arcani",
    "society": "Politica di casate nobili e corporazioni mercantili spietate",
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
      "Guardie di Marea",
      "Corporazioni Mercantili"
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
        "key": "salute",
        "label": "Salute",
        "starting": 20
      },
      {
        "key": "energia",
        "label": "Energia",
        "starting": 10
      },
      {
        "key": "resistenza",
        "label": "Resistenza",
        "starting": 15
      }
    ],
    "attributes": [
      {
        "key": "forza",
        "label": "Forza",
        "starting": 3
      },
      {
       
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `0.616 s`
- Completion tokens: `1185`
- Prompt tokens: `4393`
- Estimated cost: `$0.000594`
- Throughput: `1924.5 tok/s`, `3654.1 char/s`

```text
{
  "narrative": "La nebbia ti avvolge come un sudario umido quando varchi la Banchina delle Nebbie. L'acqua scura riflette le lanterne tremolanti di alcune case a piolo, lontane dall'orizzonte della laguna. Senti il vibrare... non un vero suono, ma un'eco, come quando una campana sommersa canta nella profondità.\n\nIl tuo corpo ricorda il peso delle mappe rubate, il sapore del sale sulle labbra, i debiti che ti hanno portato qui. Due mesi a pagamento gli interessi, e ora sei di nuovo al lavoro, con le tasche vuote e la mente piena di progetti malsani.\n\nLa banchina è affollata di scavatori di relitti che trascinano casse di legno marcio, e mercanti che controllano i carichi di sale. Una guardia di Marea ti osserva con un'aria di sospetto — sei conosciuta? Forse sì, forse no. Il tuo debito con la Casata Valcerra pesa su ogni tua mossa.\n\nIn un angolo, un vecchio mercante di reliquie ti chiama con un gesto delle dita: \"Cercavi qualcosa?\". In un altro, due mercenari discutono ad alta voce di una nave affondata vicino alle isole del nord.",
  "choices": [
    {
      "id": 1,
      "text": "Interrogare il vecchio mercante di reliquie"
    },
    {
      "id": 2,
      "text": "Seguire la guardia di Marea per cercare informazioni"
    },
    {
      "id": 3,
      "text": "Esplorare la banchina alla ricerca di lavoro"
    },
    {
      "id": 4,
      "text": "Cercare di evitare le guardie e nasconderti"
    }
  ],
  "mood": "mysterious",
  "location": "Banchina delle Nebbie",
  "scene_type": "introduction",
  "dialogue_blocks": [
    {
      "speaker": "Vecchio Mercante",
      "role": "npc",
      "text": "Cercavi qualcosa?"
    }
  ],
  "entities_mentioned": [
    {
      "name": "Vecchio Mercante",
      "type": "npc"
    },
    {
      "name": "Guardia di Mare
...[truncated]...
```

### Dialogue scene with renderer metadata

- Compat score: `85.0/100`
- Duration: `0.563 s`
- Completion tokens: `903`
- Prompt tokens: `4663`
- Estimated cost: `$0.000538`
- Throughput: `1603.9 tok/s`, `4184.6 char/s`
- Notes:
  - semantic metadata scarso sulle choices

```text
{"narrage":"Lyanna ti fissa con occhi grigi come l'acqua della bassa marea. Accarezza per un istante le dita contro il tessuto cerato del suo mantello, poi risponde a bassa voce.\n\n\"Quelle campane... risvegliano cose che non dovresti ricordare.\" Fa una pausa, il suono di una goccia che cade dal molo si mescola alle loro voci. \"Le pattuglie cercano entrambe le cose, ma so dove nascondersi i corrieri rubati.\"\n\nLa sua voce si abbassa ulteriormente, quasi un sussurro. \"Se vuoi restare viva, non mostrare la reliquia a chiunque. Nemmeno a me.\" La sua disposizione verso di te si è leggermente modificata, probabilmente perché hai mostrato interesse per le verità che lei custodisce.\n\n\"Cosa fai adesso?\" chiede, con le dita che battano per la terza volta sul tavolo di legno marcio.","choices":[{"id":1,"text":"Chiedile dove nascondersi dai Dock Wardens"},{"id":2,"text":"Le mostri la reliquia per dimostrare che è in tuo possesso"},{"id":3,"text":"Le offri una corona di sale per informazioni più dettagliate"},{"id":4,"text":"Le chiedi se sa chi ha organizzato il furto delle campane"}],"mood":"tense","location":"Old Harbor","scene_type":"dialogue","dialogue_blocks":[{"speaker":"Lyanna","role":"npc","text":"Quelle campane... risvegliano cose che non dovresti ricordare."},{"speaker":"Lyanna","role":"npc","text":"Le pattuglie cercano entrambe le cose, ma so dove nascondersi i corrieri rubati."},{"speaker":"Lyanna","role":"npc","text":"Se vuoi restare viva, non mostrare la reliquia a chiunque. Nemmeno a me."},{"speaker":"Lyanna","role":"npc","text":"Cosa fai adesso?"}],"entities_mentioned":[{"name":"Lyanna Voss","type":"npc"},{"name":"Dock Wardens","type":"faction"},{"name":"Old Harbor","type":"location"},{"name":"campane sommerse","type":"relic"}],"event_callouts":[{"ki
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `85.0/100`
- Duration: `0.660 s`
- Completion tokens: `611`
- Prompt tokens: `4430`
- Estimated cost: `$0.000447`
- Throughput: `925.5 tok/s`, `1819.1 char/s`

```text
{"narrative":"Ti lanci verso la fune laterale mentre il ponte di vetro si frantuma sotto i tuoi piedi. I lanternieri urlano, le loro catene cariche di sale incandescente sferzano l'aria. L'acqua in piena ti afferra le caviglie, tirandoti verso il canale nero.\n\nLa fune vibra violentemente. Un passo falso e saresti già inghiottita.\n\nIl sistema di gioco sta per determinare se riesci a mantenerti in equilibrio mentre il ponte collassa dietro di te.","choices":[{"id":1,"text":"[Attendi Risultato Sfida]","id":2,"text":"Spingi con le gambe e salta verso la riva opposta","id":3,"text":"Lascia cadere il sacchetto di reliquie per alleggerirti","id":4,"text":"Afferra il lanterniero più vicino e usalo come scudo"}],"mood":"desperate","location":"Stormglass Causeway","scene_type":"action_sequence","challenges":[{"type":"stat_check","stat":"agi","difficulty":75,"description":"Equilibrio sulla fune mentre il ponte collassa"}],"entities_mentioned":[{"name":"Nerea","type":"npc"},{"name":"Lanternieri","type":"npc"},{"name":"Stormglass Causeway","type":"location"}],"event_callouts":[{"kind":"action","title":"Crociera sulla fune","detail":"Sfida critica di Agilità in corso"}],"state_changes":{}}
```

### Chapter title + summary

- Compat score: `85.0/100`
- Duration: `0.487 s`
- Completion tokens: `4853`
- Prompt tokens: `574`
- Estimated cost: `$0.001299`
- Throughput: `9964.1 tok/s`, `3449.3 char/s`
- Notes:
  - Copertura limitata di eventi/entità attese

```text
{
  "title": "Whispers of the Flooded Harbor",
  "summary": "Nerea arrives at the Foggy Wharf carrying a stolen map sewn into her coat and the heavy weight of her brother’s name in a debt register. Seeking a covert passage, she negotiates with a suspicious boatman who accepts half payment but identifies the broken seal on her wrist, realizing she is fleeing pursuers. Following the trail, Nerea reaches the Tide Market, where she meets Lyanna Voss. Lyanna warns that the Tide Guards are hunting a missing bell relic, which Nerea reveals to her. Recognizing the Seal of the Salt Chorus, Lyanna guides Nerea to a flooded warehouse housing a vibrating salt altar. In a pivotal decision, Nerea touches the relic with her knife tip, causing it to partially open. This action reveals a haunting memory fragment: her brother kneeling before a black bell while guards knock at the door. The vision confirms her brother’s dangerous connection to the order. Fearing discovery, they flee immediately. During the escape, a bridge collapses, forcing Nerea to abandon a bag of salt, though Lyanna saves her from a sudden surge of water. Lyanna decides to assist Nerea in reaching the Old Harbor. The chapter closes with Nerea vowing to discover why her brother was bound to the Salt Chorus. As night descends, submerged bells begin to ring again, echoing through the waterlogged streets. A new debt is magically engraved upon Nerea’s name, signaling that her troubles have only deepened. The world shifts as the relic’s power awakens, marking her as a target for both the Guards and the Chorus. The mystery of the black bell looms large, promising further danger in the Old Harbor."
}
```

## x-ai/grok-4.1-fast

- Context: `2000000`
- Max completion: `30000`
- Prompt cost/token: `$0.000000200`
- Completion cost/token: `$0.000000500`
- Avg latency: `0.550 s`
- Avg cost: `$0.001421`
- Total benchmark cost: `$0.007106`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `0.586 s`
- Completion tokens: `2488`
- Prompt tokens: `1115`
- Estimated cost: `$0.001467`
- Throughput: `4244.6 tok/s`, `5205.0 char/s`

```text
{
  "name": "Vespera Oscura",
  "description": "Vespera è una città-laguna decadente ispirata a una Venezia sommersa da canali salati e costruita su isole nere, dove politica intrigante e culti del mare dominano tra nebbie e sussurri. La magia echeggia dalle campane sommerse, alterando menti e destini, mentre corporazioni spietate e guardie corrotte lottano per il potere. In questo mondo malinconico ma avventuroso, esplora quartieri allagati pieni di reliquie, affronta pericoli del profondo e forgia il tuo cammino tra echi perduti.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico ma Avventuroso",
  "setting": {
    "world_name": "Vespera",
    "era": "Decadenza Rinascimentale",
    "geography": "Città-laguna su isole nere e canali di sale, con quartieri allagati pieni di reliquie.",
    "magic_system": "Magia dei Suoni Sommeri: derivata dagli echi delle campane sommerse che alterano memoria e volontà; la magia ha sempre un costo personale, il sale protegge dai sussurri del mare.",
    "technology_level": "Rinascimentale (gondole, maschere, archi gotici, armi da fuoco primitive, influenzato da magia marittima)",
    "society": "Dominata da casate nobili in lotta per il potere, corporazioni mercantili spietate, religione delle campane sommerse, guardie di porto corrotte e classi come scavatori e pellegrini.",
    "rules": [
      "La magia ha sempre un costo personale.",
      "Il sale protegge dai sussurri del mare.",
      "Le campane sommerse alterano memoria e volontà.",
      "I giuramenti pubblici hanno valore legale e spirituale.",
      "Il debito non pagato può essere venduto come servitù."
    ],
    "factions": [
      "Casata Valcerra (nobili intriganti dei canali)",
      "Coro del Sale (religione delle campane sommerse)",
      "Guardie di Marea 
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `0.490 s`
- Completion tokens: `1355`
- Prompt tokens: `4278`
- Estimated cost: `$0.001533`
- Throughput: `2763.6 tok/s`, `3385.7 char/s`

```text
{
  "narrative": "La nebbia avvolge la Banchina delle Nebbie come un sudario umido, intriso dell'odore salmastro dei canali di Vespera. I ponti bassi cigolano sotto i tuoi passi incerti, mentre le banchine nere, viscide di alghe e residui di maree spezzate, si estendono verso isole lontane, collegate da fili d'acqua torbida. In lontananza, un rintocco sommesso risuona dalle campane sommerse: un suono che piega la memoria, sussurrando debiti dimenticati e fortune perdute. Hai pagato il prezzo per averlo ascoltato troppe volte, Nerea, ex corriere delle banchine, cresciuta tra mappe rubate e catene di debiti familiari.\n\nStringi in mano una pergamena stropicciata, frutto di un furto audace a un magazzino della Casata Valcerra. Otto corone di sale tintinnano nella tua borsa, appena sufficienti per un pasto o un passaggio. Il tuo debito pulsa come una ferita aperta: due corone dovute al Coro del Sale, e gli usurai non perdonano. Una figura incappucciata osserva dal ponte vicino, e un'imbarcazione fantasma emerge piano dalla foschia.",
  "choices": [
    {"id": 1, "text": "Esamina la mappa rubata con attenzione"},
    {"id": 2, "text": "Avvicinati alla figura incappucciata sul ponte"},
    {"id": 3, "text": "Dirigiti verso la locanda più vicina per raccogliere informazioni"},
    {"id": 4, "text": "Controlla la tua borsa e i debiti accumulati"}
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

- Compat score: `70.0/100`
- Duration: `0.623 s`
- Completion tokens: `1705`
- Prompt tokens: `4526`
- Estimated cost: `$0.001758`
- Throughput: `2737.5 tok/s`, `3360.5 char/s`
- Notes:
  - dialogue_blocks insufficienti o assenti
  - event_callouts assenti
  - semantic metadata scarso sulle choices

```text
{
  "narrative": "Mantieni la voce bassa, un sussurro che si mescola al ticchettio della pioggia sui moli allagati. Lyanna ti fissa, i suoi occhi stanchi che ti squadrano per un lungo istante. Tocca due volte il bordo del sacchetto di sale – un tic che conosci bene – prima di rispondere, asciutta e tagliente.\n\n\"Sotto le campane? Sussurri di debiti vecchi come Vespera stessa. Giuramenti che il sale non può più sigillare. Le pattuglie... cercano reliquie. Qualsiasi cosa luccichi di troppo. Il tuo nome non circola, Nerea, ma quella tua 'cosina' di scarso valore? Attirerebbe i Dock Wardens come mosche sul miele marcio.\"",
  "choices": [
    {"id": 1, "text": "Mostrale la reliquia per farle capire il rischio"},
    {"id": 2, "text": "Chiedile se può aiutarti a nasconderla o smerciarla"},
    {"id": 3, "text": "Proponi di dividerci e ritrovarci al Mercato della Marea"},
    {"id": 4, "text": "Insisti per sapere di più sui sussurri delle campane"}
  ],
  "mood": "tense",
  "location": "Old Harbor",
  "scene_type": "dialogue",
  "dialogue_blocks": [
    {"speaker": "Lyanna Voss", "role": "npc", "text": "\"Sotto le campane? Sussurri di debiti vecchi come Vespera stessa. Giuramenti che il sale non può più sigillare. Le pattuglie... cercano reliquie. Qualsiasi cosa luccichi di troppo. Il tuo nome non circola, Nerea, ma quella tua 'cosina' di scarso valore? Attirerebbe i Dock Wardens come mosche sul miele marcio.\""}
  ],
  "entities_mentioned": [
    {"name": "Lyanna Voss", "type": "npc"},
    {"name": "Dock Wardens", "type": "faction"}
  ],
  "state_changes": {
    "npc_disposition": {"name": "Lyanna Voss", "change": 5},
    "npc_thoughts": {"name": "Lyanna Voss", "thought": "Nerea è diretta ma mantiene la calma; potrebbe valere la pena tenerla d'occhio."},
    "np
...[truncated]...
```

### Challenge-producing action scene

- Compat score: `100.0/100`
- Duration: `0.589 s`
- Completion tokens: `1885`
- Prompt tokens: `4302`
- Estimated cost: `$0.001803`
- Throughput: `3200.9 tok/s`, `2555.6 char/s`

```text
{
  "narrative": "Schizzi verso la fune laterale con la tua agilità da corriere dei canali, il ponte di Stormglass che si frantuma in schegge di vetro nero dietro di te. L'acqua ribolle sotto, spingendo detriti contro i piloni. Le catene dei lanternieri fischiano nell'aria umida, sfiorandoti la schiena mentre afferri la corda vibrante e inizi a correre sull'instabile passaggio sospeso. Il vento gelido e le oscillazioni improvvise mettono alla prova il tuo equilibrio: un passo falso e precipiterai nel canale in piena.\n\nLa tua tenacia ti spinge avanti, ma la fune trema violentemente.",
  "choices": [
    {"id": 1, "text": "Concentrati sul ritmo dei passi e accelera"},
    {"id": 2, "text": "Usa la mano libera per aggrapparti meglio alla fune"},
    {"id": 3, "text": "Guarda indietro per valutare la distanza dai lanternieri"}
  ],
  "mood": "desperate",
  "location": "Stormglass Causeway",
  "scene_type": "chase",
  "challenges": [
    {
      "type": "dice_roll",
      "difficulty": 65,
      "description": "Mantenere l'equilibrio sulla fune instabile durante il inseguimento",
      "modifiers": [
        {"source": "Agilità (4)", "value": 20},
        {"source": "Skill Canali (lvl 2)", "value": 15},
        {"source": "Skill Corsa (lvl 1)", "value": 10},
        {"source": "Trait Tenace", "value": 10}
      ]
    }
  ],
  "state_changes": {
    "skill_xp": {"skill": "Corsa", "xp": 25},
    "skill_xp": {"skill": "Canali", "xp": 20},
    "vitals": {"stress": {"current": 5}}
  }
}
```

### Chapter title + summary

- Compat score: `85.0/100`
- Duration: `0.467 s`
- Completion tokens: `816`
- Prompt tokens: `688`
- Estimated cost: `$0.000546`
- Throughput: `1748.3 tok/s`, `3828.7 char/s`
- Notes:
  - Copertura limitata di eventi/entità attese

```text
{
  "title": "Relic's Submerged Echoes",
  "summary": "Nerea arrives at the Foggy Dock (Banchina delle Nebbie) with a stolen map sewn into her coat and her brother's name marked in a debt ledger, driven by a quest to uncover his fate. She discreetly negotiates passage with the ferryman (battelliere), who accepts half payment but notices the broken seal on her wrist, deducing she's fleeing pursuers. Following her lead, she reaches the Tide Market (Mercato della Marea), where she encounters Lyanna Voss, a key NPC who warns her of the Tide Guards searching for a vanished bell relic that appeared at dawn.\n\nNerea reveals the relic to Lyanna, who identifies its Salt Choir seal and leads her to a flooded warehouse featuring a vibrating salt altar attuned to submerged bell tolls. Probing the relic with her knife tip, Nerea unlocks a fragment of memory: her brother kneeling before a black bell as Tide Guards knock at the door, revealing his ties to the enigmatic Salt Choir.\n\nAlerted to approaching danger, Nerea decides to flee with Lyanna. Chaos ensues as a bridge collapses during their escape; Nerea loses her salt pouch but is rescued by Lyanna from a sudden flood. Grateful and committed, Lyanna pledges to guide Nerea to the Old Harbor. The chapter closes on Nerea's vow to unravel her brother's Salt Choir connection, punctuated by the resounding submerged bells and a new debt etched in her name.\n\nNo new items or abilities are gained, but the relic proves pivotal, offering visions. Locations traversed include Foggy Dock, Tide Market, flooded warehouse, and en route to Old Harbor. Major revelations center on the brother's Salt Choir involvement and the relic's memory function, heightening pursuit by Tide Guards amid a misty, tidal world of debts and secrets."
}
```

## google/gemini-2.5-flash-lite

- Context: `1048576`
- Max completion: `65535`
- Prompt cost/token: `$0.000000100`
- Completion cost/token: `$0.000000400`
- Avg latency: `0.760 s`
- Avg cost: `$0.000519`
- Total benchmark cost: `$0.002596`

### Story creation final JSON

- Compat score: `100.0/100`
- Duration: `1.355 s`
- Completion tokens: `777`
- Prompt tokens: `999`
- Estimated cost: `$0.000411`
- Throughput: `573.4 tok/s`, `1749.0 char/s`

```text
{
  "name": "Vespera: Echi Sommersi",
  "description": "Un'avventura fantasy oscura ambientata nella decadente città-laguna di Vespera, dove intrighi politici, culti marini e la malinconia si intrecciano tra canali oscuri e antiche campane sommerse.",
  "genre": "Fantasy Oscuro",
  "tone": "Malinconico, Avventuroso, Cupo",
  "setting": {
    "world_name": "Vespera",
    "era": "Epoca di Decadenza",
    "geography": "Città-laguna costruita su isole nere e canali di sale, con quartieri allagati e rovine sommerse.",
    "magic_system": "La magia deriva dal suono delle campane sommerse, che alterano memoria e volontà. Il suo utilizzo ha sempre un costo.",
    "technology_level": "Rinascimentale con elementi magici",
    "society": "Dominata da casate nobiliari in lotta per il potere, corporazioni mercantili spietate e una religione legata al mare e alle campane.",
    "rules": [
      "La magia ha sempre un costo.",
      "Il sale protegge dai sussurri del mare.",
      "Le campane sommerse alterano memoria e volontà.",
      "I giuramenti pubblici hanno valore legale e spirituale.",
      "Il debito non pagato può essere venduto come servitù."
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
      {"key": "hp", "label": "Punti Ferita", "starting": 10},
      {"key": "mana", "label": "Eco Interiore", "starting": 5},
      {"key": "stamina", "label": "Resistenza", "starting": 8}
    ],
    "attributes": [
      {"key": "str", "
...[truncated]...
```

### Narrative first turn

- Compat score: `100.0/100`
- Duration: `0.729 s`
- Completion tokens: `553`
- Prompt tokens: `4411`
- Estimated cost: `$0.000662`
- Throughput: `758.3 tok/s`, `2494.4 char/s`

```text
{
  "narrative": "Il salmastro ti morde le narici mentre l'aria umida e pesante della Banchina delle Nebbie ti accarezza il viso. Le assi di legno scricchiolano sotto i tuoi piedi, un lamento sommesso nel silenzio rotto solo dal sciabordio delle onde contro gli ormeggi decrepiti. Lanterne a olio gettano pozze di luce tremolante su cumuli di reti da pesca in disuso e barili sigillati, ma le ombre danzano, dense e impenetrabili, tra i vicoli stretti e i ponti bassi che collegano quest'isola lagunare al resto di Vespera. L'odore di pesce marcio si mescola a qualcosa di più antico, un sentore di sale puro e mistero che permea ogni cosa. Sei qui perché il debito di famiglia incombe, un'ombra più persistente della nebbia stessa, e le voci parlano di un'opportunità tra le reliquie emerse dalle profondità... o forse tra i sussurri delle campane sommerse.",
  "choices": [
    {"id": 1, "text": "Dirigiti verso la taverna \"Il Polpo Ubriaco\" per raccogliere informazioni."},
    {"id": 2, "text": "Esamina i relitti alla ricerca di qualcosa di valore."},
    {"id": 3, "text": "Cerca un membro delle Guardie di Marea per capire la situazione attuale."},
    {"id": 4, "text": "Recati al tempio del Coro del Sale per pregare per una guida."}
  ],
  "mood": "malinconico",
  "location": "Banchina delle Nebbie",
  "scene_type": "introduction",
  "dialogue_blocks": [],
  "entities_mentioned": [],
  "event_callouts": [],
  "state_changes": {
    "world_location_add": "Banchina delle Nebbie",
    "world_location_add": "Il Polpo Ubriaco",
    "world_location_add": "Mercato dei Relitti",
    "world_location_add": "Posto di Guardia delle Guardie di Marea",
    "world_location_add": "Tempio del Coro del Sale"
  },
  "challenges": [],
  "achievement_earned": null,
  "chapter_end": false,
  "c
...[truncated]...
```

### Dialogue scene with renderer metadata

- Compat score: `70.0/100`
- Duration: `0.680 s`
- Completion tokens: `446`
- Prompt tokens: `4672`
- Estimated cost: `$0.000646`
- Throughput: `655.7 tok/s`, `1890.8 char/s`
- Notes:
  - dialogue_blocks insufficienti o assenti
  - event_callouts assenti
  - semantic metadata scarso sulle choices

```text
{
  "narrative": "Parli sottovoce, la tua voce quasi inghiottita dal tamburellare della pioggia. \"Cosa hai sentito sotto le campane, Lyanna? E ti cercano per me, o per la cosa che ho?\" Gli occhi stanchi di Lyanna saettano verso il sacchetto di sale che tiene in mano, poi tornano su di te. Socchiude le labbra, esitando.",
  "choices": [
    {
      "id": 1,
      "text": "Insisti, chiedendo una risposta diretta."
    },
    {
      "id": 2,
      "text": "Mostrale la reliquia, sperando di guadagnare la sua fiducia."
    },
    {
      "id": 3,
      "text": "Interrompi la conversazione e suggerisci di muoversi."
    }
  ],
  "mood": "tense",
  "location": "Old Harbor",
  "scene_type": "dialogue",
  "dialogue_blocks": [
    {
      "speaker": "Nerea",
      "role": "protagonist",
      "text": "Cosa hai sentito sotto le campane, Lyanna? E ti cercano per me, o per la cosa che ho?"
    }
  ],
  "entities_mentioned": [
    {
      "name": "Lyanna Voss",
      "type": "npc"
    }
  ],
  "event_callouts": [],
  "state_changes": {
    "npc_disposition": {
      "name": "Lyanna Voss",
      "change": 5
    },
    "skill_xp": {
      "skill": "Canali",
      "xp": 15
    }
  },
  "challenges": [],
  "achievement_earned": null,
  "chapter_end": false,
  "chapter_title": ""
}
```

### Challenge-producing action scene

- Compat score: `65.0/100`
- Duration: `0.600 s`
- Completion tokens: `386`
- Prompt tokens: `4439`
- Estimated cost: `$0.000598`
- Throughput: `643.4 tok/s`, `1893.6 char/s`
- Notes:
  - challenges assente
  - challenge presente ma con shape debole o incompleta

```text
{
  "narrative": "Ti lanci verso la fune, il vento ululante ti sferza il volto mentre il ponte dietro di te cede con uno schianto assordante. Le catene delle lanterne sibilano nell'aria, furiose, ma la tua agilità ti permette di mantenere l'equilibrio precario. Ogni passo è una danza disperata contro la gravità e le correnti del canale. Riesci a raggiungere l'altra sponda, ansimante, mentre i detriti della passerella sprofondano nelle acque scure.",
  "choices": [
    {
      "id": 1,
      "text": "Controlla se i lanternieri ti hanno seguita."
    },
    {
      "id": 2,
      "text": "Cerca un nascondiglio tra le macerie sulla sponda."
    },
    {
      "id": 3,
      "text": "Continua a correre verso il mercato nero."
    }
  ],
  "mood": "teso",
  "location": "Stormglass Causeway",
  "state_changes": {
    "skill_xp": {
      "skill": "Corsa",
      "xp": 25
    },
    "skill_xp": {
      "skill": "Canali",
      "xp": 25
    },
    "vitals": {
      "hp": {
        "current": 8,
        "max": 10
      },
      "stress": {
        "current": 5,
        "max": 5
      }
    }
  },
  "achievement_earned": null
}
```

### Chapter title + summary

- Compat score: `65.0/100`
- Duration: `0.440 s`
- Completion tokens: `554`
- Prompt tokens: `573`
- Estimated cost: `$0.000279`
- Throughput: `1260.0 tok/s`, `4858.0 char/s`
- Notes:
  - Summary fuori target: 171 parole
  - Copertura limitata di eventi/entità attese

```text
{
  "title": "Whispers of the Salt Choir",
  "summary": "Nerea arrives at the Fog Dock with a stolen map and her brother's name on a debt ledger. She secures passage with a wary ferryman who notices her broken wrist seal, hinting at her fugitive status. Following a lead, Nerea ventures into the Tide Market where she encounters Lyanna Voss. Lyanna warns Nerea about the Tide Guards searching for a stolen bell relic. Nerea reveals the relic, and Lyanna recognizes a Salt Choir seal, leading them to a flooded warehouse. There, an old salt altar hums in sync with submerged chimes. Nerea touches the relic with her knife, triggering a fragmented memory: her brother kneeling before a black bell as guards approach. They flee as a bridge collapses, causing Nerea to lose her salt pouch, but Lyanna saves her from a flash flood and pledges to help her reach the Old Harbor. The chapter concludes with Nerea's resolve to uncover her brother's connection to the Salt Choir, the ominous sound of submerged bells, and a new debt added to her name.",
  "locations": [
    "Banchina delle Nebbie (Fog Dock)",
    "Mercato della Marea (Tide Market)",
    "Magazzino allagato (Flooded Warehouse)",
    "Old Harbor"
  ],
  "npcs": [
    "Battelliere (Ferryman)",
    "Lyanna Voss"
  ],
  "items_gained": [
    "Mappa rubata (Stolen Map)",
    "Reliquia campanaria (Bell Relic)"
  ],
  "plot_developments": [
    "Nerea is on the run and seeking information about her brother.",
    "The Tide Guards are actively searching for a specific relic.",
    "Nerea has a fragmented memory of her brother's involvement with the Salt Choir and a black bell.",
    "Lyanna Voss becomes an ally, offering assistance.",
    "Nerea incurs a new debt."
  ],
  "decisions": [
    "Negotiated passage with the ferryman.",
    "S
...[truncated]...
```

