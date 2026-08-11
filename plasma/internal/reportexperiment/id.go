package reportexperiment

import (
	"fmt"
	"sync"
)

type runIDGenerator struct {
	mu     sync.Mutex
	stem   string
	counts map[string]int
}

func newRunIDGenerator(runID string) *runIDGenerator {
	return &runIDGenerator{stem: safeIDStem(runID), counts: map[string]int{}}
}

func (generator *runIDGenerator) NewID(prefix string) string {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	generator.counts[prefix]++
	return fmt.Sprintf("%s_reportexperiment_%s_%03d", prefix, generator.stem, generator.counts[prefix])
}
