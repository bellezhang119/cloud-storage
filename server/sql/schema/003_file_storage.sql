-- +goose Up

-- Folders table
CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    UNIQUE(user_id, parent_id, name)
);

-- Files table
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    file_path TEXT NOT NULL,               
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    mime_type TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    UNIQUE(folder_id, name)
);

-- File shares table
CREATE TABLE file_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    shared_with INT REFERENCES users(id) ON DELETE CASCADE,
    permission TEXT CHECK (permission IN ('read','write','owner')),
    created_at TIMESTAMP DEFAULT now(),
    UNIQUE(file_id, shared_with)
);

-- Activity log table (optional for audit/logging)
CREATE TABLE file_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,          -- e.g. UPLOAD, DOWNLOAD, DELETE, SHARE
    details JSONB,                        -- optional extra info
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS file_activity;
DROP TABLE IF EXISTS file_shares;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS folders;
