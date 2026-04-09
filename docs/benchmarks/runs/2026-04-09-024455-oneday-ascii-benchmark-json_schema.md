# OneDay ASCII Model Benchmark

- Generated: `2026-04-09T02:33:42+02:00`
- Base URL: `https://openrouter.ai/api/v1`
- Mode: `json_schema`
- Raw JSON artifact: `2026-04-09-024455-oneday-ascii-benchmark-json_schema.json`
- Scoring: automated ASCII runtime suitability; visual taste still needs human review.

## Overall Leaderboard

| Model | Score | Success | Avg Seconds | Avg Cost | Context | Max Out | Avg Tok/s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `openai/gpt-oss-120b:free` | 100.0 | 100% | 0.685 | $0.000000 | 131072 | 131072 | 253.2 |
| `x-ai/grok-4.1-fast` | 96.0 | 100% | 0.588 | $0.001204 | 2000000 | 30000 | 3740.3 |
| `google/gemma-4-26b-a4b-it:free` | 93.0 | 100% | 0.714 | $0.000000 | 262144 | 32768 | 229.1 |
| `google/gemini-2.5-flash-lite` | 93.0 | 100% | 1.099 | $0.000073 | 1048576 | 65535 | 106.7 |
| `google/gemma-4-31b-it:free` | 92.0 | 100% | 0.969 | $0.000000 | 262144 | 32768 | 152.7 |
| `nvidia/nemotron-3-super-120b-a12b:free` | 88.0 | 100% | 9.761 | $0.000000 | 262144 | 262144 | 24.1 |
| `z-ai/glm-4.5-air:free` | 0.0 | 100% | 6.905 | $0.000000 | 131072 | 96000 | 47.7 |
| `minimax/minimax-m2.5:free` | 0.0 | 0% | 0.000 | $0.000000 | 196608 | 196608 | 0.0 |

## Best Quality / Cost

| Model | Metric | Score | Avg Seconds | Avg Cost |
| --- | ---: | ---: | ---: | ---: |
| `openai/gpt-oss-120b:free` | 100000000.000 | 100.0 | 0.685 | $0.000000 |
| `google/gemma-4-26b-a4b-it:free` | 93000000.000 | 93.0 | 0.714 | $0.000000 |
| `google/gemma-4-31b-it:free` | 92000000.000 | 92.0 | 0.969 | $0.000000 |
| `nvidia/nemotron-3-super-120b-a12b:free` | 88000000.000 | 88.0 | 9.761 | $0.000000 |
| `google/gemini-2.5-flash-lite` | 1278057.719 | 93.0 | 1.099 | $0.000073 |
| `x-ai/grok-4.1-fast` | 79728.701 | 96.0 | 0.588 | $0.001204 |
| `z-ai/glm-4.5-air:free` | 0.000 | 0.0 | 6.905 | $0.000000 |
| `minimax/minimax-m2.5:free` | 0.000 | 0.0 | 0.000 | $0.000000 |

## Best Quality / Latency

| Model | Metric | Score | Avg Seconds | Avg Cost |
| --- | ---: | ---: | ---: | ---: |
| `x-ai/grok-4.1-fast` | 163.265 | 96.0 | 0.588 | $0.001204 |
| `openai/gpt-oss-120b:free` | 145.950 | 100.0 | 0.685 | $0.000000 |
| `google/gemma-4-26b-a4b-it:free` | 130.252 | 93.0 | 0.714 | $0.000000 |
| `google/gemma-4-31b-it:free` | 94.976 | 92.0 | 0.969 | $0.000000 |
| `google/gemini-2.5-flash-lite` | 84.648 | 93.0 | 1.099 | $0.000073 |
| `nvidia/nemotron-3-super-120b-a12b:free` | 9.016 | 88.0 | 9.761 | $0.000000 |
| `z-ai/glm-4.5-air:free` | 0.000 | 0.0 | 6.905 | $0.000000 |
| `minimax/minimax-m2.5:free` | 0.000 | 0.0 | 0.000 | $0.000000 |

## Best Practical Runtime Fit

| Model | Metric | Score | Avg Seconds | Avg Cost |
| --- | ---: | ---: | ---: | ---: |
| `x-ai/grok-4.1-fast` | 163.265 | 96.0 | 0.588 | $0.001204 |
| `openai/gpt-oss-120b:free` | 145.950 | 100.0 | 0.685 | $0.000000 |
| `google/gemma-4-26b-a4b-it:free` | 130.252 | 93.0 | 0.714 | $0.000000 |
| `google/gemma-4-31b-it:free` | 94.976 | 92.0 | 0.969 | $0.000000 |
| `google/gemini-2.5-flash-lite` | 84.648 | 93.0 | 1.099 | $0.000073 |
| `nvidia/nemotron-3-super-120b-a12b:free` | 9.016 | 88.0 | 9.761 | $0.000000 |
| `z-ai/glm-4.5-air:free` | 0.000 | 0.0 | 6.905 | $0.000000 |
| `minimax/minimax-m2.5:free` | 0.000 | 0.0 | 0.000 | $0.000000 |

## openai/gpt-oss-120b:free

- Context: `131072`
- Max completion: `131072`
- Prompt cost/token: `$0.000000000`
- Completion cost/token: `$0.000000000`
- Avg latency: `0.685 s`
- Avg cost: `$0.000000`
- Total benchmark cost: `$0.000000`

### Major location reveal

- Score: `100.0/100`
- Duration: `0.884 s`
- Output shape: `11 lines`, `max width 66`
- Completion tokens: `344`
- Prompt tokens: `354`
- Estimated cost: `$0.000000`
- Throughput: `389.0 tok/s`, `843.6 char/s`

```text
{"ascii_art":"      ^^^      ^^^      ^^^      ^^^      ^^^      ^^^\n    /|||\\    /|||\\    /|||\\    /|||\\    /|||\\    /|||\\\n   /_|||_\\  /_|||_\\  /_|||_\\  /_|||_\\  /_|||_\\  /_|||_\\\n  //=====\\\\//=====\\\\//=====\\\\//=====\\\\//=====\\\\//=====\\\\\n ||  * *  ||||  * *  ||||  * *  ||||  * *  ||||  * *  ||\n ||  \\_/  ||||  \\_/  ||||  \\_/  ||||  \\_/  ||||  \\_/  ||\n ||_______||||_______||||_______||||_______||||_______||\n  \\\\_____//  \\\\_____//  \\\\_____//  \\\\_____//  \\\\_____//  \\\\_____//\n   \\\\   //    \\\\   //    \\\\   //    \\\\   //    \\\\   //    \\\\   //\n    \\\\ //      \\\\ //      \\\\ //      \\\\ //      \\\\ //      \\\\ //\n     \\/        \\/        \\/        \\/        \\/        \\/"}
```

### Neon signage

- Score: `100.0/100`
- Duration: `0.757 s`
- Output shape: `12 lines`, `max width 29`
- Completion tokens: `189`
- Prompt tokens: `347`
- Estimated cost: `$0.000000`
- Throughput: `249.7 tok/s`, `453.2 char/s`

```text
{"ascii_art":"   .-.-.   .-.-.   .-.-.   .-.-.\n  ( T A L L O N E )\n   \\   |   /   |   /   |\n    \\  |  /    |  /    |\n     \\ | /     | /     |\n      \\|/      |/      |\n   .-.-.-.-.-.-.-.-.-.-.-.\n  /   G L I T T E R   \\\n |   D E C A D E N C E |\n  \\   N E O N   L I G H T /\n   '-._   _.-'   '-._   _.-'\n       ''''         ''''"}
```

### Terminal screen

- Score: `100.0/100`
- Duration: `0.931 s`
- Output shape: `11 lines`, `max width 32`
- Completion tokens: `154`
- Prompt tokens: `332`
- Estimated cost: `$0.000000`
- Throughput: `165.4 tok/s`, `413.5 char/s`

```text
{"ascii_art":"+------------------------------+\n|  ACCESS DENIED               |\n|  --------------------------  |\n|  > [C]ONNECT   [R]ESET      |\n|  > [L]OGS    [S]TATUS       |\n|  > [H]ELP    [Q]UIT         |\n|  --------------------------  |\n|  ERROR: AUTHORIZATION FAIL   |\n|  CODE: 0xC0R0-4C0R0          |\n|  CONTACT: ADMIN@CHORUS.SYS   |\n+------------------------------+"}
```

### Ritual circle / diagram

- Score: `100.0/100`
- Duration: `0.497 s`
- Output shape: `10 lines`, `max width 19`
- Completion tokens: `125`
- Prompt tokens: `337`
- Estimated cost: `$0.000000`
- Throughput: `251.4 tok/s`, `428.3 char/s`

```text
{"ascii_art":"          .-.-.\n        .'     '.\n       /  * * *  \\\n      |  *  o  *  |\n      | *   |   * |\n       \\  * | *  /\n        '.  |  .'\n          '-.-'\n        /   |   \\\n       /____|____\\\n"}
```

### Map fragment

- Score: `100.0/100`
- Duration: `0.551 s`
- Output shape: `12 lines`, `max width 19`
- Completion tokens: `135`
- Prompt tokens: `335`
- Estimated cost: `$0.000000`
- Throughput: `245.1 tok/s`, `432.2 char/s`

```text
{"ascii_art":"   ~~~ Old Harbor ~~~\n   ~~~~~~~~~~~~~~~~\n   |   |   |   |\n   | D |---| C |\n   |   |   |   |\n   |   |   |   |\n   |   |   |   |\n   |   |   |   |\n   |   |   |   |\n   |   |   |   |\n   |   |   |   |\n   S---*---F---G"}
```

### Iconic artifact reveal

- Score: `100.0/100`
- Duration: `0.494 s`
- Output shape: `10 lines`, `max width 12`
- Completion tokens: `108`
- Prompt tokens: `336`
- Estimated cost: `$0.000000`
- Throughput: `218.7 tok/s`, `293.7 char/s`

```text
{"ascii_art":"   .-''''-.\n  /  .-.  \\\n |  /   \\ |\n | |  *  | |\n |  \\   / |\n  \\  '-'  /\n   '-.__-'\n   /||||\\\n  /_||||_\\\n   `----'"}
```

## x-ai/grok-4.1-fast

- Context: `2000000`
- Max completion: `30000`
- Prompt cost/token: `$0.000000200`
- Completion cost/token: `$0.000000500`
- Avg latency: `0.588 s`
- Avg cost: `$0.001204`
- Total benchmark cost: `$0.007224`

### Major location reveal

- Score: `100.0/100`
- Duration: `0.678 s`
- Output shape: `12 lines`, `max width 26`
- Completion tokens: `3030`
- Prompt tokens: `503`
- Estimated cost: `$0.001616`
- Throughput: `4471.2 tok/s`, `469.3 char/s`

```text
 {"ascii_art":"         *     *     *\n        /\\   /\\   /\\\n       /  \\_/  \\_/  \\\n          []     []   \n         []  []  []  []\n        /         \\   \n       ~~~~~~~~~~~~~~~\n      /\/\/\/\/\/\/\/\/\n     =====================\n    /\\\/    \\/    /\\\/\n   elevated  devotional\n  arcologies    shrines"}
```

### Neon signage

- Score: `100.0/100`
- Duration: `0.567 s`
- Output shape: `8 lines`, `max width 39`
- Completion tokens: `2829`
- Prompt tokens: `495`
- Estimated cost: `$0.001513`
- Throughput: `4989.0 tok/s`, `604.9 char/s`

```text
 {"ascii_art": "     _______________________________ \n    /         TALLONE D'ORO          \\\n   /      *** CLUB ***               \\\n  |                                   |\n  |       IL CULTO OSTENTATO          |\n  |         * * * * * * *             |\n   \\_________________________________/ \n      ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~  "}
```

### Terminal screen

- Score: `100.0/100`
- Duration: `0.609 s`
- Output shape: `11 lines`, `max width 26`
- Completion tokens: `1341`
- Prompt tokens: `480`
- Estimated cost: `$0.000767`
- Throughput: `2202.2 tok/s`, `532.1 char/s`

```text
 {"ascii_art": "+------------------------+\n|   CORO DEL SALE        |\n|                        |\n|     ACCESSO            |\n|       NEGATO           |\n|                        |\n|  ! DEVOTIONAL LOCK !   |\n|                        |\n| *Praise the Eternal*   |\n|                        |\n+========================+"}
```

### Ritual circle / diagram

- Score: `80.0/100`
- Duration: `0.543 s`
- Output shape: `13 lines`, `max width 36`
- Completion tokens: `1455`
- Prompt tokens: `488`
- Estimated cost: `$0.000825`
- Throughput: `2677.1 tok/s`, `859.3 char/s`
- Notes:
  - Fuori bounds utili: lines=13 width=36

```text
 {"ascii_art":"          .................\n       ..........................\n     ....    +++++++++++++    ....\n    ...   ++                ++   ...\n   ..  +  .     .-..-.     .  +  ..\n  . + .  ..   (     )   ..  . + .\n ..++.. .     |BEL|     . ..++..\n  . + .  ..   (     )   ..  . + .\n   ..  +  .     '-..-'     .  +  ..\n    ...   ++                ++   ...\n     ....    +++++++++++++    ....\n       ..........................\n          ..............."}
```

### Map fragment

- Score: `100.0/100`
- Duration: `0.546 s`
- Output shape: `10 lines`, `max width 18`
- Completion tokens: `1856`
- Prompt tokens: `482`
- Estimated cost: `$0.001024`
- Throughput: `3400.7 tok/s`, `348.1 char/s`

```text
 {"ascii_art":"       .-.\n      / ~ \\\n+----| ~ |----+--+\n     /~~~~~\\   |\n    |  |||  |   |\n     \\_____/    |\n        |       |\n       / \\      |\n      X   X-----+\n       \\_/"}
```

### Iconic artifact reveal

- Score: `100.0/100`
- Duration: `0.588 s`
- Output shape: `7 lines`, `max width 15`
- Completion tokens: `2766`
- Prompt tokens: `482`
- Estimated cost: `$0.001479`
- Throughput: `4701.4 tok/s`, `221.0 char/s`

```text
 {"ascii_art":"     .-====-.\n    /        \\\n   |          |\n   |    ***   |\n   |          |\n    \\________/\n     .. .. .."}
```

## google/gemma-4-26b-a4b-it:free

- Context: `262144`
- Max completion: `32768`
- Prompt cost/token: `$0.000000000`
- Completion cost/token: `$0.000000000`
- Avg latency: `0.714 s`
- Avg cost: `$0.000000`
- Total benchmark cost: `$0.000000`

### Major location reveal

- Score: `80.0/100`
- Duration: `0.720 s`
- Output shape: `13 lines`, `max width 36`
- Completion tokens: `273`
- Prompt tokens: `294`
- Estimated cost: `$0.000000`
- Throughput: `379.2 tok/s`, `697.2 char/s`
- Notes:
  - Fuori bounds utili: lines=13 width=36

```text
{
  "ascii_art": "      .   .  .      .   .  .      \n    _|_  _|_  _|_  _|_  _|_  _|_   \n   |   ||   ||   ||   ||   ||   |  \n  /|___||___||___||___||___||___|\\ \n |  _  _  _  _  _  _  _  _  _  _  |\n | | || || || || || || || || || | |\n | |_||_||_||_||_||_||_||_||_||_| |\n |/  /  /  /  /  /  /  /  /  /  /  |\n/  /  /  /  /  /  /  /  /  /  /  / \\\n| [][][][][][][][][][][][][][][] |\n|  _  _  _  _  _  _  _  _  _  _  | \n|_( )_( )_( )_( )_( )_( )_( )_( )_| \n  ' ' ' ' ' ' ' ' ' ' ' ' ' ' ' ' "
}
```

### Neon signage

- Score: `100.0/100`
- Duration: `0.738 s`
- Output shape: `11 lines`, `max width 31`
- Completion tokens: `154`
- Prompt tokens: `290`
- Estimated cost: `$0.000000`
- Throughput: `208.6 tok/s`, `529.5 char/s`

```text
{
"ascii_art": "  .__________________________.  \n |  /\\  /\\  /\\  /\\  /\\  /\\  |  \n | (  )(  )(  )(  )(  )(  ) |  \n |  \\/  \\/  \\/  \\/  \\/  \\/  |  \n |  ______________________  |  \n | |                      | |  \n | |   TALLONE D'ORO     | |  \n | |______________________| |  \n |  ~~~~~~~~~~~~~~~~~~~~~~  |  \n |   * GLORIA IN LUX *     |  \n |__________________________|  "
}
```

### Terminal screen

- Score: `100.0/100`
- Duration: `0.702 s`
- Output shape: `11 lines`, `max width 27`
- Completion tokens: `106`
- Prompt tokens: `272`
- Estimated cost: `$0.000000`
- Throughput: `151.0 tok/s`, `452.9 char/s`

```text
{
"ascii_art": " [!] ACCESS DENIED [!]\n _________________________\n| [X] AUTH_ERR: 0x000    |\n|                         |\n|  { DEVOTIO_FAILURE }   |\n|                         |\n|  > RE-ENTER LITURGY    |\n|  > PURGE_SESSION       |\n|_________________________|\n  ||                 ||\n  || [!] [!] [!] [!] ||"
}
```

### Ritual circle / diagram

- Score: `100.0/100`
- Duration: `0.702 s`
- Output shape: `11 lines`, `max width 19`
- Completion tokens: `130`
- Prompt tokens: `283`
- Estimated cost: `$0.000000`
- Throughput: `185.1 tok/s`, `344.5 char/s`

```text
{
  "ascii_art": "      .---.      \n    /  _|_  \\    \n   |  ( o )  |   \n    \\  _|_  /    \n  .---'---'---.  \n /  /|  / \\  |\\  \\\n|  | | / o \\ | |  |\n \\  \\| \\___/ |/  /\n  '--'--' '--'--' \n   . . . . . .   \n  ' ' ' ' ' ' '  "
}
```

### Map fragment

- Score: `80.0/100`
- Duration: `0.690 s`
- Output shape: `13 lines`, `max width 29`
- Completion tokens: `204`
- Prompt tokens: `274`
- Estimated cost: `$0.000000`
- Throughput: `295.5 tok/s`, `603.9 char/s`
- Notes:
  - Fuori bounds utili: lines=13 width=29

```text
{
  "ascii_art": "  [ OLD HARBOR SUB-LEVELS ]\n  ~~~~~~~~~~~~~~~~~~~~~~~~~\n  |  /--\\      |      /--\\ |\n  | |    |  [X] |     |    ||\n  |  \\--/  /\\  |  /\\  \\--/ |\n  |_______/  \\_| /  \\_|____|\n  |  | |  |  | | |  |  |  |\n  |  | |  |  | | |  |  |  |\n  |  | |  |  | | |  |  |  |\n  |  | |  |  | | |  |  |  |\n  |  \\_\\__\\_\\_\\_\\_\\__\\_/  |\n  |  [SAFE ROUTE: ===]    |\n  ~~~~~~~~~~~~~~~~~~~~~~~~~"
}
```

### Iconic artifact reveal

- Score: `100.0/100`
- Duration: `0.734 s`
- Output shape: `11 lines`, `max width 21`
- Completion tokens: `114`
- Prompt tokens: `277`
- Estimated cost: `$0.000000`
- Throughput: `155.3 tok/s`, `373.4 char/s`

```text
{
"ascii_art": "      .-------.      \n    .'         '.    \n   /   _______   \\   \n  |   /  ___  \\   |  \n  |  |  ( @ )  |  |  \n  |   \\_______/   |  \n   \\      |      /   \n    '.____|____.'    \n     |         |     \n     |  [###]  |     \n     '---------'     "
}
```

## google/gemini-2.5-flash-lite

- Context: `1048576`
- Max completion: `65535`
- Prompt cost/token: `$0.000000100`
- Completion cost/token: `$0.000000400`
- Avg latency: `1.099 s`
- Avg cost: `$0.000073`
- Total benchmark cost: `$0.000437`

### Major location reveal

- Score: `80.0/100`
- Duration: `1.219 s`
- Output shape: `12 lines`, `max width 79`
- Completion tokens: `124`
- Prompt tokens: `292`
- Estimated cost: `$0.000079`
- Throughput: `101.7 tok/s`, `799.1 char/s`
- Notes:
  - Fuori bounds utili: lines=12 width=79

```text
{"ascii_art":"                                    .-----.                                    \n                                   /       \\                                   \n                                  |  .-.  |                                  \n                                  |  '-'  |                                  \n                                  \\ ----- /                                  \n                                   `-----'                                   \n                                    / | \\                                    \n                                   /  |  \\                                   \n                                  /   |   \\                                  \n                                 /    |    \\                                 \n                                /_____|_____\\                                \n                               |___________|                                \n"}
```

### Neon signage

- Score: `100.0/100`
- Duration: `1.071 s`
- Output shape: `11 lines`, `max width 22`
- Completion tokens: `140`
- Prompt tokens: `288`
- Estimated cost: `$0.000085`
- Throughput: `130.7 tok/s`, `249.3 char/s`

```text
{
  "ascii_art": "  .--~~~~~~~~~--.\n /   TALLONE D'ORO   \\\n|  *  *  *  *  *  *  |\n|  ~  ~  ~  ~  ~  ~  |\n|  *  *  *  *  *  *  |\n \\  ~  ~  ~  ~  ~  ~ /\n  `--~~~~~~~~~--'\n     / / | | \\ \\\n    / /  | |  \\ \\\n   / /   | |   \\ \\\n  /_/____|_|____\\_\\\n"
}
```

### Terminal screen

- Score: `100.0/100`
- Duration: `1.217 s`
- Output shape: `10 lines`, `max width 42`
- Completion tokens: `94`
- Prompt tokens: `270`
- Estimated cost: `$0.000065`
- Throughput: `77.2 tok/s`, `376.3 char/s`

```text
{"ascii_art":"\n+----------------------------------------+\n|  ACCESS DENIED                         |\n|                                        |\n|  >> INVOCATION FAILED                  |\n|  >> SANCTITY VIOLATED                  |\n|                                        |\n|  [ PRAYER PROTOCOL OFFLINE ]           |\n|                                        |\n|  [ RETURN TO THE LITURGY ]             |\n+----------------------------------------+\n"}
```

### Ritual circle / diagram

- Score: `100.0/100`
- Duration: `0.522 s`
- Output shape: `9 lines`, `max width 20`
- Completion tokens: `79`
- Prompt tokens: `281`
- Estimated cost: `$0.000060`
- Throughput: `151.2 tok/s`, `375.1 char/s`

```text
{
  "ascii_art": "\n      _.--\"--._\n    .'          `.\n   /   O      O   \\\n  |    \\  ^^  /    |\n  \\     `----'     /\n   `. _______ .'\n     //_____\\\\\n    (( ____ ))\n     `------'\n"
}
```

### Map fragment

- Score: `80.0/100`
- Duration: `1.366 s`
- Output shape: `13 lines`, `max width 72`
- Completion tokens: `163`
- Prompt tokens: `272`
- Estimated cost: `$0.000092`
- Throughput: `119.4 tok/s`, `714.7 char/s`
- Notes:
  - Fuori bounds utili: lines=13 width=72

```text
{"ascii_art":"\n+----------------------------------------------------------------------+\n|                                                                      |\n|   ~~~   Old Harbor Tunnels   ~~~                                     |\n|                                                                      |\n|   [ ]----[ ]----[ ]----[ ]                                          |\n|    |       |       |     |                                           |\n|   [ ]----[ ]----[ ]----[ ]                                          |\n|    |       |       |     |                                           |\n|   [ ]----[ ]----[ ]----[ ]                                          |\n|    |       |       |     |                                           |\n|   [ ]----[ ]----[ ]----[ ]                                          |\n|                                                                      |\n+----------------------------------------------------------------------+\n"}
```

### Iconic artifact reveal

- Score: `100.0/100`
- Duration: `1.200 s`
- Output shape: `11 lines`, `max width 15`
- Completion tokens: `72`
- Prompt tokens: `275`
- Estimated cost: `$0.000056`
- Throughput: `60.0 tok/s`, `160.0 char/s`

```text
{"ascii_art":"    .------.\n   /  _  _  \\\n  |  (.)(.)  |\n  |    <     |\n  |  '----'  |\n   \\________/\n  /__________\\\n |____________|\n |____________|\n |____________|\n  '----------'"}
```

## google/gemma-4-31b-it:free

- Context: `262144`
- Max completion: `32768`
- Prompt cost/token: `$0.000000000`
- Completion cost/token: `$0.000000000`
- Avg latency: `0.969 s`
- Avg cost: `$0.000000`
- Total benchmark cost: `$0.000000`

### Major location reveal

- Score: `100.0/100`
- Duration: `1.162 s`
- Output shape: `12 lines`, `max width 51`
- Completion tokens: `250`
- Prompt tokens: `294`
- Estimated cost: `$0.000000`
- Throughput: `215.1 tok/s`, `493.9 char/s`

```text
```json
{"ascii_art": "      ___          ___          ___\n     |   |        |   |        |   |\n  ___|   |___  ___|   |___  ___|   |___\n /   \\___/   \\/   \\___/   \\/   \\___/   \\\n|  _   _   _  |  _   _   _  |  _   _   _  |\n| | | | | | | || | | | | | || | | | | | ||\n| |_| |_| |_| || |_| |_| |_|| |_| |_| |_||\n|      ___      |      ___      |      ___ |\n|     /   \\     |     /   \\     |     /   \\|\n|____|  ( )  |__|____|  ( )  |__|____|  ( ) ||\n  |||  \\___/  |||  |||  \\___/  |||  |||  \\___/ ||\n  |||         |||  |||         |||  |||         |||"}
```
```

### Neon signage

- Score: `80.0/100`
- Duration: `0.917 s`
- Output shape: `13 lines`, `max width 53`
- Completion tokens: `144`
- Prompt tokens: `290`
- Estimated cost: `$0.000000`
- Throughput: `157.1 tok/s`, `727.5 char/s`
- Notes:
  - Fuori bounds utili: lines=13 width=53

```text
```json
{"ascii_art": "   ___________________________________________\n  /                                           \\\n |   *  .  .  T A L L O N E   D ' O R O  .  .  *  |\n |  _________________________________________  |\n | |                                         | |\n | |   ~  S I N G O L A R I T A   D I V I N A  ~  | |\n | |_________________________________________| |\n |                                             |\n  \\___________________________________________/\n          ||                           ||\n          ||                           ||\n     _____||___________________________||_____\n    /                                         \\"}
```
```

### Terminal screen

- Score: `80.0/100`
- Duration: `0.950 s`
- Output shape: `14 lines`, `max width 47`
- Completion tokens: `141`
- Prompt tokens: `272`
- Estimated cost: `$0.000000`
- Throughput: `148.5 tok/s`, `733.9 char/s`
- Notes:
  - Fuori bounds utili: lines=14 width=47

```text
```json
{"ascii_art": " ___________________________________________\n| [!] SYSTEM ALERT : ARCHIVIO DEL CORO [!]   |\n|-------------------------------------------|\n|                                           |\n|   >> ACCESS DENIED                         |\n|   >> STATUS: UNWORTHY                      |\n|                                           |\n|       /\\          \"The Salt claims          |\n|      /  \\          what the flesh           |\n|     /____\\         refuses to give.\"       |\n|                                           |\n|-------------------------------------------|\n| [ LOCKOUT ACTIVE ] [ PRAYER REQUIRED ]    |\n|___________________________________________|"}
```
```

### Ritual circle / diagram

- Score: `90.0/100`
- Duration: `0.919 s`
- Output shape: `11 lines`, `max width 27`
- Completion tokens: `137`
- Prompt tokens: `283`
- Estimated cost: `$0.000000`
- Throughput: `149.1 tok/s`, `381.9 char/s`
- Notes:
  - Contiene caratteri non ASCII

```text
```json
{"ascii_art": "          .---.          \n       ./       \\.       \n    ./     ___     \\.    \n   /   .---'   '---.   \\   \n  /   /   \\  |  /   \\   \\  \n |   |  -- (🔔) --  |   | \n  \\   \\   /  |  \\   /   /  \n   \\   '---. _ .---'   /   \n    \\.     '---'     ./    \n       \\.         ./       \n          '---'          "}
```
```

### Map fragment

- Score: `100.0/100`
- Duration: `0.937 s`
- Output shape: `12 lines`, `max width 12`
- Completion tokens: `101`
- Prompt tokens: `274`
- Estimated cost: `$0.000000`
- Throughput: `107.8 tok/s`, `199.6 char/s`

```text
```json
{"ascii_art": "  ___[H]___\n  |   |   |\n  |--[X]-- |\n  |   |   |\n  |   |   |\n  |--[O]-- |\n  |   |   |\n  |   |   |\n  |--[X]-- |\n  |   |   |\n  |___|___|\n  (~~~~~~~) "}
```
```

### Iconic artifact reveal

- Score: `100.0/100`
- Duration: `0.931 s`
- Output shape: `12 lines`, `max width 32`
- Completion tokens: `129`
- Prompt tokens: `277`
- Estimated cost: `$0.000000`
- Throughput: `138.6 tok/s`, `458.8 char/s`

```text
```json
{"ascii_art": "     _____________________\n    /                     \\\n   /   ___________________   \\\n  /   /                   \\   \\\n |   |       _______       |   |\n |   |      /   |   \\      |   |\n |   |     |    |    |     |   |\n |   |      \\___|___/      |   |\n |   |          |          |   |\n |   |       ___|___       |   |\n  \\   \\___________________/   /\n   \\_________________________/"}
```
```

## nvidia/nemotron-3-super-120b-a12b:free

- Context: `262144`
- Max completion: `262144`
- Prompt cost/token: `$0.000000000`
- Completion cost/token: `$0.000000000`
- Avg latency: `9.761 s`
- Avg cost: `$0.000000`
- Total benchmark cost: `$0.000000`

### Major location reveal

- Score: `100.0/100`
- Duration: `10.036 s`
- Output shape: `8 lines`, `max width 33`
- Completion tokens: `331`
- Prompt tokens: `323`
- Estimated cost: `$0.000000`
- Throughput: `33.0 tok/s`, `28.9 char/s`

```text
{"ascii_art":"      /\\      /\\      /\\      \n     /  \\    /  \\    /  \\     \n    /____\\  /____\\  /____\\    \n   | [] []| | [] []| | [] []|   \n   |  <>  | |  <>  | |  <>  |   \n   |~~~~~~| |~~~~~~| |~~~~~~|   \n   \\______/  \\______/  \\______/  \n      ||      ||      ||     "}
```

### Neon signage

- Score: `100.0/100`
- Duration: `10.241 s`
- Output shape: `4 lines`, `max width 31`
- Completion tokens: `191`
- Prompt tokens: `318`
- Estimated cost: `$0.000000`
- Throughput: `18.7 tok/s`, `14.0 char/s`

```text
{
  "ascii_art": "  ______________________ \n /  T A L L O N E   D' O R O \\\n|  \"LUCE DIVINA\" (flickering) |\n \\______________________/ "
}
```

### Terminal screen

- Score: `90.0/100`
- Duration: `10.042 s`
- Output shape: `5 lines`, `max width 28`
- Completion tokens: `188`
- Prompt tokens: `295`
- Estimated cost: `$0.000000`
- Throughput: `18.7 tok/s`, `13.1 char/s`
- Notes:
  - Art poco visiva: troppo testo o pochi segni grafici

```text
{"ascii_art":"[ACCESS DENIED]\n> LITURGIC PROTOCOL\n> SALT CHOIR ARCHIVE\n> AUTHORIZATION REQUIRED\n> [DEVOTIONAL INDUSTRIAL UI]\n"}
```

### Ritual circle / diagram

- Score: `70.0/100`
- Duration: `9.915 s`
- Output shape: `1 lines`, `max width 303`
- Completion tokens: `320`
- Prompt tokens: `306`
- Estimated cost: `$0.000000`
- Throughput: `32.3 tok/s`, `32.5 char/s`
- Notes:
  - Fuori bounds utili: lines=1 width=303
  - La forma non sembra ideale per il tipo di scena richiesto

```text
{"ascii_art":"   . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . ."}
```

### Map fragment

- Score: `70.0/100`
- Duration: `8.295 s`
- Output shape: `1 lines`, `max width 45`
- Completion tokens: `195`
- Prompt tokens: `299`
- Estimated cost: `$0.000000`
- Throughput: `23.5 tok/s`, `8.7 char/s`
- Notes:
  - Fuori bounds utili: lines=1 width=45
  - La forma non sembra ideale per il tipo di scena richiesto

```text
{
  "ascii_art": "  ~~~~~~\\n  +---+~\\n  | S |~\\n  +---+~\\n~~~~~~~"
}
```

### Iconic artifact reveal

- Score: `100.0/100`
- Duration: `10.036 s`
- Output shape: `6 lines`, `max width 11`
- Completion tokens: `188`
- Prompt tokens: `302`
- Estimated cost: `$0.000000`
- Throughput: `18.7 tok/s`, `9.0 char/s`

```text
{"ascii_art":"  _______\n /       \\\n|  .---.  |\n|  | O |  |\n|  '---'  |\n \\_______/"}
```

## z-ai/glm-4.5-air:free

- Context: `131072`
- Max completion: `96000`
- Prompt cost/token: `$0.000000000`
- Completion cost/token: `$0.000000000`
- Avg latency: `6.905 s`
- Avg cost: `$0.000000`
- Total benchmark cost: `$0.000000`

### Major location reveal

- Score: `0.0/100`
- Duration: `4.839 s`
- Output shape: `0 lines`, `max width 0`
- Completion tokens: `400`
- Prompt tokens: `300`
- Estimated cost: `$0.000000`
- Throughput: `82.7 tok/s`, `0.0 char/s`
- Notes:
  - Nessun payload JSON parseabile trovato

```text

```

### Neon signage

- Score: `0.0/100`
- Duration: `6.284 s`
- Output shape: `0 lines`, `max width 0`
- Completion tokens: `260`
- Prompt tokens: `294`
- Estimated cost: `$0.000000`
- Throughput: `41.4 tok/s`, `0.0 char/s`
- Notes:
  - Nessun payload JSON parseabile trovato

```text

```

### Terminal screen

- Score: `0.0/100`
- Duration: `10.041 s`
- Output shape: `0 lines`, `max width 0`
- Completion tokens: `260`
- Prompt tokens: `278`
- Estimated cost: `$0.000000`
- Throughput: `25.9 tok/s`, `0.0 char/s`
- Notes:
  - Nessun payload JSON parseabile trovato

```text

```

### Ritual circle / diagram

- Score: `0.0/100`
- Duration: `6.503 s`
- Output shape: `0 lines`, `max width 0`
- Completion tokens: `320`
- Prompt tokens: `285`
- Estimated cost: `$0.000000`
- Throughput: `49.2 tok/s`, `0.0 char/s`
- Notes:
  - Nessun payload JSON parseabile trovato

```text

```

### Map fragment

- Score: `0.0/100`
- Duration: `5.791 s`
- Output shape: `0 lines`, `max width 0`
- Completion tokens: `300`
- Prompt tokens: `279`
- Estimated cost: `$0.000000`
- Throughput: `51.8 tok/s`, `0.0 char/s`
- Notes:
  - Nessun payload JSON parseabile trovato

```text

```

### Iconic artifact reveal

- Score: `0.0/100`
- Duration: `7.976 s`
- Output shape: `0 lines`, `max width 0`
- Completion tokens: `280`
- Prompt tokens: `282`
- Estimated cost: `$0.000000`
- Throughput: `35.1 tok/s`, `0.0 char/s`
- Notes:
  - Nessun payload JSON parseabile trovato

```text

```

## minimax/minimax-m2.5:free

- Context: `196608`
- Max completion: `196608`
- Prompt cost/token: `$0.000000000`
- Completion cost/token: `$0.000000000`
- Avg latency: `0.000 s`
- Avg cost: `$0.000000`
- Total benchmark cost: `$0.000000`

### Major location reveal

- Error: `status 429: {"error":{"message":"Provider returned error","code":429,"metadata":{"raw":"minimax/minimax-m2.5:free is temporarily rate-limited upstream. Please retry shortly, or add your own key to accumulate your rate limits: https://openrouter.ai/settings/integrations","provider_name":"OpenInference","is_byok":false}},"user_id":"user_2u8YY3bZHzxLNvlJyTR56FsBKB5"}`

### Neon signage

- Error: `context deadline exceeded`

### Terminal screen

- Error: `status 429: {"error":{"message":"Provider returned error","code":429,"metadata":{"raw":"minimax/minimax-m2.5:free is temporarily rate-limited upstream. Please retry shortly, or add your own key to accumulate your rate limits: https://openrouter.ai/settings/integrations","provider_name":"OpenInference","is_byok":false}},"user_id":"user_2u8YY3bZHzxLNvlJyTR56FsBKB5"}`

### Ritual circle / diagram

- Error: `context deadline exceeded`

### Map fragment

- Error: `status 429: {"error":{"message":"Provider returned error","code":429,"metadata":{"raw":"minimax/minimax-m2.5:free is temporarily rate-limited upstream. Please retry shortly, or add your own key to accumulate your rate limits: https://openrouter.ai/settings/integrations","provider_name":"OpenInference","is_byok":false}},"user_id":"user_2u8YY3bZHzxLNvlJyTR56FsBKB5"}`

### Iconic artifact reveal

- Error: `status 429: {"error":{"message":"Provider returned error","code":429,"metadata":{"raw":"minimax/minimax-m2.5:free is temporarily rate-limited upstream. Please retry shortly, or add your own key to accumulate your rate limits: https://openrouter.ai/settings/integrations","provider_name":"OpenInference","is_byok":false}},"user_id":"user_2u8YY3bZHzxLNvlJyTR56FsBKB5"}`

