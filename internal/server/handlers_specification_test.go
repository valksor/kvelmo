package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valksor/go-mehrhof/internal/server/views"
)

func TestSpecItemData_Structure(t *testing.T) {
	// Test that SpecItemData has the correct fields
	spec := views.SpecItemData{
		Number:      1,
		Name:        "specification-1",
		Title:       "Test Specification",
		Status:      "draft",
		Description: "Test description content",
		Component:   "backend",
		CreatedAt:   "2026-01-26 15:00",
		CompletedAt: "",
	}

	assert.Equal(t, 1, spec.Number)
	assert.Equal(t, "specification-1", spec.Name)
	assert.Equal(t, "Test Specification", spec.Title)
	assert.Equal(t, "draft", spec.Status)
	assert.Equal(t, "Test description content", spec.Description)
	assert.Equal(t, "backend", spec.Component)
	assert.Equal(t, "2026-01-26 15:00", spec.CreatedAt)
	assert.Equal(t, "", spec.CompletedAt)
}

func TestSpecificationsData_Structure(t *testing.T) {
	// Test that SpecificationsData has the correct fields
	data := views.SpecificationsData{
		Items: []views.SpecItemData{
			{
				Number:      1,
				Name:        "specification-1",
				Title:       "Test",
				Status:      "draft",
				Description: "Content",
			},
		},
		Total:    1,
		Done:     0,
		Progress: 0.0,
	}

	assert.Equal(t, 1, data.Total)
	assert.Equal(t, 0, data.Done)
	assert.Equal(t, 0.0, data.Progress)
	assert.Len(t, data.Items, 1)
	assert.Equal(t, "specification-1", data.Items[0].Name)
}
