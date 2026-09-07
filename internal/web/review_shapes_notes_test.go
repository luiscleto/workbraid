package web

import (
	"testing"
	"workbraid/internal/architecture"
)

func TestShapeReviewDistinguishesDefaultFromNotVisible(t *testing.T) {
	diamond := "diamond"
	projection := func(shape *string) snapshotProjectionResponse {
		return snapshotProjectionResponse{Diagrams: []diagramResponse{{ID: "diagram", Shapes: []architecture.NodeShapeChange{{ComponentID: "component", Shape: shape}}}}}
	}
	absent := snapshotProjectionResponse{}
	for _, tc := range []struct {
		name                       string
		before, with               snapshotProjectionResponse
		beforeVisible, withVisible bool
	}{
		{"disappears", projection(&diamond), absent, true, false},
		{"appears", absent, projection(&diamond), false, true},
		{"default remains visible", projection(&diamond), projection(nil), true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shapes, _ := compareShapesNotes(tc.before, tc.with)
			if len(shapes) != 1 || shapes[0].BeforeVisible != tc.beforeVisible || shapes[0].WithVisible != tc.withVisible {
				t.Fatalf("incorrect visibility: %+v", shapes)
			}
		})
	}
}
