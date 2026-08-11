package reportworkflow

import (
	"fmt"
	"sort"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partplan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

func runLimited(count int, limit int, run func(int) error) error {
	if count == 0 {
		return nil
	}
	if limit > count {
		limit = count
	}
	sem := make(chan struct{}, limit)
	errs := make([]error, count)
	var wg sync.WaitGroup
	for index := 0; index < count; index++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(index int) {
			defer wg.Done()
			defer func() { <-sem }()
			errs[index] = run(index)
		}(index)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func sectionIndexKey(partIndex, sectionNumber int) sectionIndex {
	return sectionIndex{part: partIndex, section: sectionNumber}
}

func hasRecoveredSection(progress longFormProgress, partIndex int) bool {
	for index := range progress.sections {
		if index.part == partIndex {
			return true
		}
	}
	return false
}

func recoveredSectionsForPart(progress longFormProgress, partIndex int, sectionCount int) ([]sectiondraft.Draft, error) {
	recovered := make([]sectiondraft.Draft, 0, sectionCount)
	for sectionIndex := 0; sectionIndex < sectionCount; sectionIndex++ {
		if draft, exists := progress.sections[sectionIndexKey(partIndex, sectionIndex)]; exists {
			recovered = append(recovered, draft)
		}
	}
	if len(recovered) != 0 && len(recovered) != sectionCount {
		return nil, fmt.Errorf("%w: recovered long-form part has partial section provenance", producterror.ErrConflict)
	}
	return recovered, nil
}

func sortedPartPlanIndexes(values map[int]partplan.Output) []int {
	indexes := make([]int, 0, len(values))
	for index := range values {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return indexes
}
