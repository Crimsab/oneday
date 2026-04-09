# Summary 14.1: Home Archive and Story-Scoped Achievements

Added a home-surface `Achievements` entry point from the main menu plus a story archive summary layer that loads each run's protagonist, location, turn, and unlocked achievements without starting the narrative session.

The new achievement browser supports the requested accordion flow on the home surface: stories list first, each story expands to its unlocked achievements, and selecting one opens a focused detail view with description, rarity, timing, and story metadata. Inside an active run, `/achievements` now reuses the same browser model but stays filtered to the current story only.
