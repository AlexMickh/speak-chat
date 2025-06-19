CREATE SCHEMA IF NOT EXISTS chat;

CREATE TABLE IF NOT EXISTS chat.chats(
    id UUID PRIMARY KEY,
    name TEXT,
    description TEXT,
    owner_id TEXT,
    chat_image_url TEXT,
    participants_id TEXT ARRAY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
