package missionusage

import (
	"math"
	"sort"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

func markUnavailable(summary *Summary) {
	summary.UsageUnavailableCount++
	summary.UsagePartial = true
}

func addUsage(summary *Summary, usage agentusage.ProviderUsage) bool {
	fields := []struct {
		target *int64
		value  int
	}{
		{&summary.InputTokens, usage.InputTokens},
		{&summary.CachedInputTokens, usage.CachedInputTokens},
		{&summary.UncachedInputTokens, usage.UncachedInputTokens},
		{&summary.OutputTokens, usage.OutputTokens},
		{&summary.ReasoningOutputTokens, usage.ReasoningOutputTokens},
		{&summary.TotalTokens, usage.TotalTokens},
	}
	for _, field := range fields {
		if field.value < 0 || int64(field.value) > math.MaxInt64-*field.target {
			return false
		}
	}
	for _, field := range fields {
		*field.target += int64(field.value)
	}
	return true
}

func percentiles(values []int64) Percentiles {
	if len(values) == 0 {
		return Percentiles{}
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return Percentiles{
		P50: values[nearestRank(len(values), 50)],
		P90: values[nearestRank(len(values), 90)],
		Max: values[len(values)-1],
	}
}

func nearestRank(length int, percentile int) int {
	return (length*percentile+99)/100 - 1
}
