ALTER TABLE comments 
ADD COLUMN parent_id UUID REFERENCES comments(id);

CREATE INDEX idx_comments_parent_id ON comments(parent_id);