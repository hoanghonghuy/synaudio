ALTER TABLE stories DROP CONSTRAINT IF EXISTS fk_stories_current_content_profile_version;
DROP TABLE IF EXISTS story_content_profile_versions;
