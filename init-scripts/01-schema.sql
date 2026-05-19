CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE book (
    -- Unique Identifiers
    id BIGSERIAL PRIMARY KEY,
    isbn_13 VARCHAR(13) UNIQUE CONSTRAINT check_isbn_13_length CHECK (length(isbn_13) = 13),
    isbn_10 VARCHAR(10) UNIQUE CONSTRAINT check_isbn_10_length CHECK (length(isbn_10) = 10),
    -- Core Metadata
    title VARCHAR(255) NOT NULL,
    subtitle VARCHAR(255),
    -- Flattened Relationships (No external tables needed)
    authors TEXT[] NOT NULL, -- Native array supporting multiple authors: '{"Author A", "Author B"}'
    publisher VARCHAR(255),
    -- Details
    description TEXT,
    language_code VARCHAR(10) DEFAULT 'en', -- Standard ISO 639-1
    page_count INTEGER CONSTRAINT check_positive_pages CHECK (page_count > 0),
    publication_date DATE,
    edition VARCHAR(50),
    -- Flexibility Format (For categories, tags, or retail pricing metadata)
    metadata JSONB DEFAULT '{}'::jsonb, -- Stores arbitrary key-values (e.g., {"price": 19.99, "genre": "Sci-Fi"})
    -- Media & Auditing
    cover_image_url VARCHAR(2048),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Performance Optimization Indexes
CREATE INDEX idx_book_title ON book (title);

CREATE INDEX idx_book_authors ON book USING gin (authors);

-- Optimized for searching within arrays
CREATE INDEX idx_book_metadata ON book USING gin (metadata);

-- Optimized for deep JSON queries
Create TABLE book_emb (
    book_id BIGSERIAL,
    embedding vector (768),
    PRIMARY KEY (book_id),
    FOREIGN KEY (book_id) REFERENCES book (id)
);

CREATE INDEX ON book_emb USING hnsw (embedding vector_ip_ops)
WITH
    (m = 32, ef_construction = 128);