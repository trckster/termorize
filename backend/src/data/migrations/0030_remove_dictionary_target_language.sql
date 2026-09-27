-- Edition is already snapshotted on each job and determines its source format
-- and language filter. Keep 0029 unchanged for installations that applied it.
ALTER TABLE dictionaries DROP COLUMN IF EXISTS target_language;
ALTER TABLE dictionary_import_jobs DROP COLUMN IF EXISTS target_language;
