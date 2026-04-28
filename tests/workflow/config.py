"""Workflow test configuration."""

from pathlib import Path

PROJECT_ROOT = Path(__file__).parent.parent.parent


class TestConfig:
    """General test configuration."""

    BACKEND_BASE_URL = "http://localhost:5000"
    FIRECRAWL_BASE_URL = "http://192.168.5.27:3002"

    BACKEND_TIMEOUT = 60
    FIRECRAWL_TIMEOUT = 60
    AI_TIMEOUT = 120

    DATABASE_PATH = str(PROJECT_ROOT / "backend-go" / "rss_reader.db")

    SCHEDULER_INTERVALS = {
        "auto_refresh": 60,
        "firecrawl": 300,
        "ai_summary": 3600,
        "preference_update": 1800,
    }

    FIRECRAWL_CONCURRENCY = 3
    AI_SUMMARY_SINGLE_THREAD = True
    MAX_COMPLETION_RETRIES = 3

    TEST_ARTICLE_URL = "https://sspai.com/post/105308"
    TEST_FEED_URL = "https://sspai.com/feed"

    FIRECRAWL_STATUSES = ["pending", "processing", "completed", "failed"]
    CONTENT_STATUSES = ["incomplete", "pending", "complete", "failed"]
    REFRESH_STATUSES = ["idle", "refreshing", "error"]

    MAX_CONTENT_LENGTH = 50000
    MIN_CONTENT_LENGTH = 100


class DatabaseConfig:
    """Database fixture defaults."""

    TEST_CATEGORY = {
        "name": "Test Category",
        "slug": "test-category",
        "icon": "folder",
        "color": "#8b5cf6",
    }

    TEST_FEED = {
        "title": "Test Feed",
        "url": "https://sspai.com/feed",
        "refresh_interval": 60,
        "firecrawl_enabled": True,
        "content_completion_enabled": True,
        "max_completion_retries": 3,
    }


class MockConfig:
    """Mock service payloads."""

    MOCK_FIRECRAWL_RESPONSE = {
        "success": True,
        "data": {
            "markdown": "# Test Article\n\nThis is test content.",
            "html": "<h1>Test Article</h1><p>This is test content.</p>",
            "metadata": {
                "title": "Test Article",
                "description": "Test description",
            },
        },
    }

    MOCK_AI_SUMMARY = {
        "one_sentence": "This is a test article.",
        "key_points": ["Point 1", "Point 2", "Point 3"],
        "takeaways": ["Takeaway 1", "Takeaway 2"],
        "tags": ["test", "example"],
    }
