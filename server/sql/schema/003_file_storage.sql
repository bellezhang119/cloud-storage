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
    user_id INT REFERENCES users(id),
    name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    mime_type TEXT,
    is_trashed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    deleted_at TIMESTAMP,
    UNIQUE(folder_id, name)
);

-- File shares table
CREATE TABLE file_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    shared_with INT REFERENCES users(id),
    permission TEXT CHECK (permission IN ('read','write','owner')),
    created_at TIMESTAMP DEFAULT now()
);

-- Activity log table
CREATE TABLE file_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID REFERENCES files(id),
    user_id INT REFERENCES users(id),
    action TEXT NOT NULL,
    details JSONB,
    created_at TIMESTAMP DEFAULT now()
);

-- +goose Down

DROP TABLE file_activity;
DROP TABLE file_shares;
DROP TABLE files;
DROP TABLE folders;
