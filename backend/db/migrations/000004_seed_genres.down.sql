DELETE FROM genres WHERE slug IN (
    'fantasy', 'science-fiction', 'romance', 'mystery', 'thriller',
    'horror', 'adventure', 'drama', 'comedy', 'historical-fiction',
    'literary-fiction', 'young-adult'
);
