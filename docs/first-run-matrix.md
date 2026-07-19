# First-run and portability matrix

`make first-run-matrix` runs the automated first-run proof matrix in bounded
slices. The initial CLI slice creates a fresh temporary workspace for every
run, supplies empty OneDay configuration/database paths, and runs only the
existing setup, doctor, fake-narrator turn, save, and restore contracts.

No real provider, credential, or external network is used. The runner starts
Go with module and checksum downloads disabled, and each command has a
five-minute default deadline. Set `ONEDAY_MATRIX_TIMEOUT_SECONDS` to a positive
integer to change that deadline. The temporary workspace is removed on exit.

The CLI proof covers:

- setup choices from empty configuration and text-only safe defaults;
- canonical doctor output, including private-path and fake-provider-failure
  redaction;
- a first playable canonical action through the existing fake narrator; and
- save/restore persistence through the in-process game service.

This is not a live-provider or packaged-desktop smoke test. Those boundaries
remain explicit: live calls require separate authorized credentials, while
desktop package behavior is validated by the platform packaging workflow rather
than this CLI slice.
