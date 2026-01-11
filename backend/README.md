# RSS Reader Backend API

Flask-based REST API for RSS feed management with SQLite database.

## Features

- **Category Management**: Create, read, update, and delete feed categories
- **Feed Management**: Subscribe to RSS feeds with automatic parsing
- **Article Management**: Read articles, track read status and favorites
- **OPML Support**: Import and export feeds in OPML format
- **Full-text Search**: Search articles by title and description
- **Filtering**: Filter articles by feed, category, read status, and favorites

## Installation

1. Create a virtual environment:
```bash
python -m venv venv
```

2. Activate the virtual environment:
```bash
# Windows
venv\Scripts\activate

# macOS/Linux
source venv/bin/activate
```

3. Install dependencies:
```bash
pip install -r requirements.txt
```

## Running the Application

1. Start the Flask server:
```bash
python app.py
```

The API will be available at `http://localhost:5000`

## API Endpoints

### Categories

- `GET /api/categories` - List all categories
- `POST /api/categories` - Create a new category
- `PUT /api/categories/<id>` - Update a category
- `DELETE /api/categories/<id>` - Delete a category

### Feeds

- `GET /api/feeds` - List all feeds (with pagination)
- `POST /api/feeds` - Create a new feed (automatically imports articles)
- `PUT /api/feeds/<id>` - Update a feed
- `DELETE /api/feeds/<id>` - Delete a feed
- `POST /api/feeds/fetch` - Fetch RSS feed metadata from URL

### Articles

- `GET /api/articles` - List articles with filters (pagination, search, read status)
- `GET /api/articles/<id>` - Get a single article
- `PUT /api/articles/<id>` - Update article (read/favorite status)
- `PUT /api/articles/bulk-update` - Bulk update articles

### OPML Import/Export

- `POST /api/import-opml` - Import feeds from OPML file
- `GET /api/export-opml` - Export feeds to OPML format

## Database Models

### Category
- `id` (Integer, Primary Key)
- `name` (String, Required, Unique)
- `slug` (String, Required, Unique)
- `icon` (String, Default: 'folder')
- `color` (String, Default: '#6366f1')
- `description` (Text)
- `created_at` (DateTime)

### Feed
- `id` (Integer, Primary Key)
- `title` (String, Required)
- `description` (Text)
- `url` (String, Required, Unique)
- `category_id` (Integer, Foreign Key)
- `icon` (String, Default: 'rss')
- `color` (String, Default: '#8b5cf6')
- `last_updated` (DateTime)
- `created_at` (DateTime)

### Article
- `id` (Integer, Primary Key)
- `feed_id` (Integer, Foreign Key, Required)
- `title` (String, Required)
- `description` (Text)
- `content` (Text)
- `link` (String)
- `pub_date` (DateTime)
- `author` (String)
- `read` (Boolean, Default: False)
- `favorite` (Boolean, Default: False)
- `created_at` (DateTime)

## Example Usage

### Create a Category
```bash
curl -X POST http://localhost:5000/api/categories \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Technology",
    "icon": "microchip",
    "color": "#3b82f6",
    "description": "Tech news and blogs"
  }'
```

### Add a Feed
```bash
curl -X POST http://localhost:5000/api/feeds \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/feed.xml",
    "category_id": 1,
    "icon": "rss",
    "color": "#8b5cf6"
  }'
```

### Get Articles
```bash
curl "http://localhost:5000/api/articles?page=1&per_page=20&read=false&favorite=true"
```

### Update Article Read Status
```bash
curl -X PUT http://localhost:5000/api/articles/1 \
  -H "Content-Type: application/json" \
  -d '{
    "read": true
  }'
```

### Import OPML
```bash
curl -X POST http://localhost:5000/api/import-opml \
  -F "file=@feeds.opml"
```

## Configuration

Edit `config.py` to customize:
- Database path
- CORS origins
- Feed update interval
- Pagination settings

## Error Handling

All endpoints return JSON responses with the following structure:

Success:
```json
{
  "success": true,
  "data": { ... }
}
```

Error:
```json
{
  "success": false,
  "error": "Error message"
}
```

## Technologies Used

- **Flask** - Web framework
- **SQLAlchemy** - ORM for database operations
- **Flask-CORS** - CORS support
- **feedparser** - RSS/Atom feed parsing
- **SQLite** - Database

## License

MIT
