# Summary 36.4 — Story-pack authoring and fairness evals

Story packs now decode as typed YAML/JSON and are rejected when identifiers,
stats, difficulty profiles, consequence budgets, challenge kinds, or required
answers are invalid. Packs can author a genre-neutral stat schema, named
difficulty/accessibility profiles, tagged challenge pools with cooldowns, and
versioned outcome policies. The noir example exercises every authoring hook
while explicitly disabling combat.

The shared host exposes an autoplay path used by a versioned golden corpus and
standalone `oneday-minigame-eval` quality gate. Twelve deterministic cases cover
success and failure across deduction, pattern, bidding, courtroom, comedy, and
negotiation. The report requires all eight audit concerns, exact expected
degrees, replay equality, authoritative outcomes, and a bounded success-rate
band; the checked corpus passes 12/12 with a 0.50 success rate.

Go, Rust, and TypeScript consume the same player-safe minigame v1 fixture, which
deliberately omits authoritative answers. The visual selection E2E mock was also
made stateful and now waits for the mutation response, eliminating a polling
race that could overwrite undo/redo state during verification.
