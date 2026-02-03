import os

class Config:
    """Application configuration"""

    # Base directory
    BASE_DIR = os.path.abspath(os.path.dirname(__file__))

    # Database
    SQLALCHEMY_DATABASE_URI = f'sqlite:///{os.path.join(BASE_DIR, "rss_reader.db")}'
    SQLALCHEMY_TRACK_MODIFICATIONS = False
    # Connection pool settings to prevent connection timeout
    # Increase pool size for concurrent background threads
    SQLALCHEMY_ENGINE_OPTIONS = {
        'pool_size': 10,  # 增加连接池大小从默认的 5 到 10
        'max_overflow': 20,  # 增加溢出连接数从默认的 10 到 20
        'pool_timeout': 60,  # 增加超时时间从默认的 30 秒到 60 秒
        'pool_recycle': 3600,  # 连接回收时间（秒），避免长时间连接
        'pool_pre_ping': True,  # 连接前检查连接是否有效
    }

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
