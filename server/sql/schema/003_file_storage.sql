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
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    file_path TEXT NOT NULL,               
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    mime_type TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    UNIQUE(user_id, folder_id, name)
);

-- File shares table
CREATE TABLE file_shares (
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    shared_user_id INT REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now(),
    UNIQUE(file_id, shared_user_id)
);

-- Folder shares table
CREATE TABLE folder_shares (
    folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    shared_user_id INT REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now(),
    UNIQUE(folder_id, shared_user_id)
);

-- Indexes for performance
CREATE INDEX idx_file_shares_user ON file_shares(shared_user_id);
CREATE INDEX idx_folder_shares_user ON folder_shares(shared_user_id);
CREATE INDEX idx_files_folder ON files(folder_id);
CREATE INDEX idx_folders_parent ON folders(parent_id);

-- +goose Down
DROP TABLE IF EXISTS file_shares;
DROP TABLE IF EXISTS folder_shares;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS folders;