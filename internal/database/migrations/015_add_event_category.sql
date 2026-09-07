-- Per-event category, exposed as the iCal CATEGORIES property.
-- Existing rows default to 'qjelt': before this column existed the feed
-- hardcoded that category on every VEVENT, so the default keeps the feed
-- byte-identical until each event is recategorised from the admin form.
ALTER TABLE events ADD COLUMN category TEXT NOT NULL DEFAULT 'qjelt';
