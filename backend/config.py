import os

class Config:
    """Application configuration"""

    # Base directory
    BASE_DIR = os.path.abspath(os.path.dirname(__file__))

    # Database
    SQLALCHEMY_DATABASE_URI = f'sqlite:///{os.path.join(BASE_DIR, "rss_reader.db")}'
    SQLALCHEMY_TRACK_MODIFICATIONS = False

    # CORS
    CORS_ORIGINS = ['http://localhost:3000', 'http://localhost:5173', 'http://localhost:8080']

    # Feed update interval (in hours)
    FEED_UPDATE_INTERVAL = 1

    # Auto-refresh scheduler check interval (in seconds)
    # How often to check for feeds that need refreshing
    AUTO_REFRESH_CHECK_INTERVAL = 60

    # Auto-summary scheduler interval (in seconds)
    # How often to auto-generate AI summaries
    AUTO_SUMMARY_INTERVAL = 3600

    # Pagination
    DEFAULT_PAGE_SIZE = 20
    MAX_PAGE_SIZE = 100
