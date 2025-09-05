-- Пользователи
CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       name TEXT NOT NULL,
                       email TEXT UNIQUE NOT NULL,
                       password_hash TEXT NOT NULL,
                       created_at TIMESTAMP DEFAULT NOW()
);

-- Чаты
CREATE TABLE chats (
                       id SERIAL PRIMARY KEY,
                       name TEXT NOT NULL,
                       type TEXT NOT NULL CHECK (type IN ('public', 'private')),
                       created_at TIMESTAMP DEFAULT NOW()
);

-- Сообщения
CREATE TABLE messages (
                          id SERIAL PRIMARY KEY,
                          chat_id INT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
                          user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          text TEXT NOT NULL,
                          created_at TIMESTAMP DEFAULT NOW()
);

-- Связь пользователей и чатов
CREATE TABLE chat_users (
                            chat_id INT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
                            user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                            joined_at TIMESTAMP DEFAULT NOW(),
                            PRIMARY KEY (chat_id, user_id)
);
