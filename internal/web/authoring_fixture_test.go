package web

import (
	"context"
	"fmt"
	"workbraid/internal/architecture"
)

// Fixtures author through the same preparation path as synchronized mutations,
// and check that its final facts replay without initialization.
func prepareTestCandidate(manager *architecture.Manager, ctx context.Context, base architecture.Snapshot, changes []architecture.ComponentChange, composition architecture.CandidateComposition) (architecture.Candidate, error) {
	candidate, err := manager.PrepareCandidate(ctx, base, changes, &composition)
	if err != nil {
		return architecture.Candidate{}, err
	}
	replay, err := manager.ConstructCandidate(ctx, base, changes, composition)
	if err != nil {
		return architecture.Candidate{}, err
	}
	if candidate.Tree() != replay.Tree() {
		return architecture.Candidate{}, fmt.Errorf("fixture facts do not replay exactly")
	}
	return candidate, nil
}
