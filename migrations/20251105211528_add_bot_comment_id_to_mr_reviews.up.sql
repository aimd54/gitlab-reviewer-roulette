-- Add bot_comment_id field to track the bot's comment for updates
ALTER TABLE mr_reviews ADD COLUMN bot_comment_id INTEGER;

-- Add index for faster lookups
CREATE INDEX idx_mr_reviews_bot_comment_id ON mr_reviews(bot_comment_id);

-- Add comment explaining the field
COMMENT ON COLUMN mr_reviews.bot_comment_id IS 'GitLab note ID of the bot comment for this review (allows updating instead of creating new comments)';
