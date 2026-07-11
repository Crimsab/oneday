package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/crimsab/oneday/internal/engine"
)

func main() {
	corpusPath := flag.String("corpus", "internal/engine/testdata/minigame-evals.json", "path to the versioned minigame evaluation corpus")
	flag.Parse()
	corpus, err := engine.LoadMiniGameEvalCorpus(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load corpus: %v\n", err)
		os.Exit(1)
	}
	report := engine.RunMiniGameEvalCorpus(*corpus)
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode report: %v\n", err)
		os.Exit(1)
	}
	if !report.Passed() {
		os.Exit(1)
	}
}
