package ical

import (
	"strings"
	"testing"
	"time"

	"github.com/toulibre/libreregistration/internal/models"
)

func TestRenderCalendarEmitsURLAndCategory(t *testing.T) {
	events := []models.Event{{
		ID:          "abc123",
		Title:       "QJeLT de juin",
		Slug:        "2026-06-04-qjelt",
		Location:    "Toulouse",
		Description: "Repas mensuel",
		Category:    "qjelt",
		EventDate:   time.Date(2026, 6, 4, 18, 30, 0, 0, time.UTC),
	}}

	var b strings.Builder
	if err := RenderCalendar(&b, events, "Toulibre", "https://evenements.toulibre.org"); err != nil {
		t.Fatalf("RenderCalendar returned error: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		"URL:https://evenements.toulibre.org/event/2026-06-04-qjelt\r\n",
		"CATEGORIES:qjelt\r\n",
		"SUMMARY:QJeLT de juin\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderCalendarOmitsURLWithoutBaseURL(t *testing.T) {
	events := []models.Event{{ID: "abc123", Title: "QJeLT", Slug: "2026-06-04-qjelt", EventDate: time.Now()}}

	var b strings.Builder
	if err := RenderCalendar(&b, events, "", ""); err != nil {
		t.Fatalf("RenderCalendar returned error: %v", err)
	}
	if strings.Contains(b.String(), "URL:") {
		t.Errorf("URL property should be omitted when baseURL is empty\n---\n%s", b.String())
	}
}

func TestRenderCalendarEmitsPerEventCategory(t *testing.T) {
	events := []models.Event{
		{ID: "a", Title: "Rencontre Blender", Slug: "blender", Category: "atelier", EventDate: time.Now()},
		{ID: "b", Title: "Permanence GNU/Linux", Slug: "permanence", Category: "permanence", EventDate: time.Now()},
	}

	var b strings.Builder
	if err := RenderCalendar(&b, events, "", ""); err != nil {
		t.Fatalf("RenderCalendar returned error: %v", err)
	}
	out := b.String()

	for _, want := range []string{"CATEGORIES:atelier\r\n", "CATEGORIES:permanence\r\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "CATEGORIES:qjelt") {
		t.Errorf("qjelt must no longer be emitted for non-qjelt events\n---\n%s", out)
	}
}

func TestRenderCalendarOmitsEmptyCategory(t *testing.T) {
	events := []models.Event{{ID: "a", Title: "Sans cat\u00e9gorie", Slug: "x", EventDate: time.Now()}}

	var b strings.Builder
	if err := RenderCalendar(&b, events, "", ""); err != nil {
		t.Fatalf("RenderCalendar returned error: %v", err)
	}
	if strings.Contains(b.String(), "CATEGORIES:") {
		t.Errorf("CATEGORIES should be omitted for an uncategorised event\n---\n%s", b.String())
	}
}
