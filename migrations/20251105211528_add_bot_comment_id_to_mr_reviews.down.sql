-- Remove bot_comment_id field and index
DROP INDEX IF EXISTS idx_mr_reviews_bot_comment_id;
ALTER TABLE mr_reviews DROP COLUMN IF EXISTS bot_comment_id;
