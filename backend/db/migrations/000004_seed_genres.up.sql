-- Seed baseline genres (idempotent via ON CONFLICT DO NOTHING).

INSERT INTO genres (id, slug, name) VALUES
    (gen_random_uuid(), 'fantasy', 'Fantasy'),
    (gen_random_uuid(), 'science-fiction', 'Science Fiction'),
    (gen_random_uuid(), 'romance', 'Romance'),
    (gen_random_uuid(), 'mystery', 'Mystery'),
    (gen_random_uuid(), 'thriller', 'Thriller'),
    (gen_random_uuid(), 'horror', 'Horror'),
    (gen_random_uuid(), 'adventure', 'Adventure'),
    (gen_random_uuid(), 'drama', 'Drama'),
    (gen_random_uuid(), 'comedy', 'Comedy'),
    (gen_random_uuid(), 'historical-fiction', 'Historical Fiction'),
    (gen_random_uuid(), 'literary-fiction', 'Literary Fiction'),
    (gen_random_uuid(), 'young-adult', 'Young Adult')
ON CONFLICT (slug) DO NOTHING;
