// Package ical renders models.Event values as iCalendar (RFC 5545) text.
package ical

import (
	"io"
	"strings"
	"time"

	"github.com/toulibre/libreregistration/internal/models"
)

const (
	prodID    = "-//LibreRegistration//EN"
	timeStamp = "20060102T150405Z"
)

// RenderCalendar writes a complete VCALENDAR document containing one VEVENT
// per entry in events. calName is exposed as X-WR-CALNAME so calendar clients
// display a friendly name; pass "" to omit it.
func RenderCalendar(w io.Writer, events []models.Event, calName string) error {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:" + prodID + "\r\n")
	if calName != "" {
		b.WriteString("X-WR-CALNAME:" + escape(calName) + "\r\n")
	}
	now := time.Now().UTC().Format(timeStamp)
	for _, e := range events {
		writeEvent(&b, e, now)
	}
	b.WriteString("END:VCALENDAR\r\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func writeEvent(b *strings.Builder, e models.Event, now string) {
	b.WriteString("BEGIN:VEVENT\r\n")
	b.WriteString("UID:" + e.ID + "@libreregistration\r\n")
	b.WriteString("DTSTAMP:" + now + "\r\n")
	b.WriteString("DTSTART:" + e.EventDate.UTC().Format(timeStamp) + "\r\n")
	b.WriteString("DTEND:" + e.EventDate.Add(2*time.Hour).UTC().Format(timeStamp) + "\r\n")
	b.WriteString("SUMMARY:" + escape(e.Title) + "\r\n")
	if e.Location != "" {
		b.WriteString("LOCATION:" + escape(e.Location) + "\r\n")
	}
	if e.Description != "" {
		b.WriteString("DESCRIPTION:" + escape(e.Description) + "\r\n")
	}
	b.WriteString("END:VEVENT\r\n")
}

func escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
