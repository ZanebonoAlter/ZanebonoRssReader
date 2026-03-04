"""Firecrawl集成测试配置"""


class TestConfig:
    """测试配置 - 支持多功能扩展"""
    
    # Firecrawl服务配置
    FIRECRAWL_BASE_URL = "http://192.168.5.27:3002"
    FIRECRAWL_TIMEOUT = 60
    
    # 后端服务配置
    BACKEND_BASE_URL = "http://localhost:5000"
    BACKEND_TIMEOUT = 60
    
    # 数据库配置
    DATABASE_PATH = "backend-go/rss_reader.db"
    
    # 测试文章配置
    TEST_ARTICLE_ID = 4
    TEST_ARTICLE_URL = "https://sspai.com/post/105308"
    
    # ============ 扩展配置区域 ============
    # 未来的去噪功能
    DENOISE_ENABLED = False
    DENOISE_CONFIG = {
        "remove_ads": True,
        "remove_navigation": True,
        "min_content_length": 100
    }
    
    # 未来的其他处理功能
    PROCESSING_PIPELINE = ["crawl"]  # 可扩展为 ["crawl", "denoise", "summarize"]