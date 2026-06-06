package usecase

import (
	"testing"

	"project-srv/internal/model"
)

func TestUnarchiveProjectStatus(t *testing.T) {
	next, ok := unarchiveProjectStatus(model.ProjectStatusArchived)
	if !ok {
		t.Fatal("expected ARCHIVED to be eligible for unarchive")
	}
	if next != model.ProjectStatusPending {
		t.Fatalf("unexpected unarchive status: got %s want %s", next, model.ProjectStatusPending)
	}

	for _, status := range []model.ProjectStatus{
		model.ProjectStatusPending,
		model.ProjectStatusActive,
		model.ProjectStatusPaused,
	} {
		if next, ok := unarchiveProjectStatus(status); ok {
			t.Fatalf("expected %s to be ineligible, got next status %s", status, next)
		}
	}
}
