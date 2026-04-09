# Summary 16.2: Committed-Turn Memory Semantics

RAG summarization now treats committed turns as zero-based canonical history. Empty summary state returns `-1` instead of pretending turn `0` was already summarized, so the first real summary window correctly starts at turn `0`.

The summarizer and RAG tests were updated to use committed-turn semantics consistently, preventing first-window skips and off-by-one drift in later summary ranges.
