package diff

import (
	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

// Diff compares two context packs and produces a drift report.
func Diff(storeRoot string, hashA string, hashB string) (*DriftReport, error) {
	a, err := pack.LoadPack(storeRoot, hashA)
	if err != nil {
		return nil, err
	}
	b, err := pack.LoadPack(storeRoot, hashB)
	if err != nil {
		return nil, err
	}

	report := &DriftReport{
		PackHashA: store.ShortHash(a.Hash, 12),
		PackHashB: store.ShortHash(b.Hash, 12),
	}

	// Same pack = no drift
	if a.Hash == b.Hash {
		return report, nil
	}

	// Run all comparisons
	report.Entries = append(report.Entries, ComparePrompts(a, b)...)
	report.Entries = append(report.Entries, CompareSteps(a, b)...)
	report.Entries = append(report.Entries, CompareOutputs(a, b)...)

	report.HasDrift = len(report.Entries) > 0
	return report, nil
}
