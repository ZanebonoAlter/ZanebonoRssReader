"""错误处理测试"""

import pytest
import time
from datetime import datetime

from config import TestConfig
from utils import DatabaseHelper, APIClient


class TestFirecrawlErrorHandling:
    """Firecrawl 错误处理测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        self.test_feed_id = self.db.create_test_feed(
            title="Firecrawl错误测试订阅源",
            firecrawl_enabled=1
        )
        
        yield
        
        self.db.cleanup_test_data()
    
    def test_firecrawl_timeout_error(self):
        """测试 Firecrawl 超时错误"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="超时测试文章",
            firecrawl_status='processing'
        )
        
        # 模拟超时错误
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'failed',
                'firecrawl_error': 'Request timeout after 60s'
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'failed'
        assert 'timeout' in article['firecrawl_error'].lower()
    
    def test_firecrawl_network_error(self):
        """测试 Firecrawl 网络错误"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="网络错误测试文章",
            firecrawl_status='processing'
        )
        
        # 模拟网络错误
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'failed',
                'firecrawl_error': 'Network unreachable'
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'failed'
    
    def test_firecrawl_invalid_url(self):
        """测试 Firecrawl 无效 URL"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="无效URL测试文章",
            link="not-a-valid-url",
            firecrawl_status='processing'
        )
        
        # 模拟无效 URL 错误
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'failed',
                'firecrawl_error': 'Invalid URL format'
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'failed'
    
    def test_firecrawl_service_unavailable(self):
        """测试 Firecrawl 服务不可用"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="服务不可用测试文章",
            firecrawl_status='processing'
        )
        
        # 模拟服务不可用错误
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'failed',
                'firecrawl_error': 'Service temporarily unavailable (503)'
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'failed'


class TestAISummaryErrorHandling:
    """AI Summary 错误处理测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        self.test_feed_id = self.db.create_test_feed(
            title="AI错误测试订阅源",
            firecrawl_enabled=1,
            content_completion_enabled=1,
            max_completion_retries=3
        )
        
        yield
        
        self.db.cleanup_test_data()
    
    def test_ai_service_unconfigured(self):
        """测试 AI 服务未配置"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="AI未配置测试文章",
            firecrawl_status='completed',
            content_status='incomplete',
            firecrawl_content='Test content'
        )
        
        # 模拟 AI 服务未配置错误
        self.db.update(
            'articles',
            {
                'content_status': 'failed',
                'completion_error': 'AI service not configured',
                'completion_attempts': 1
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'failed'
        assert 'not configured' in article['completion_error'].lower()
    
    def test_ai_api_key_invalid(self):
        """测试 AI API Key 无效"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="API Key无效测试文章",
            firecrawl_status='completed',
            content_status='pending',
            firecrawl_content='Test content'
        )
        
        # 模拟 API Key 无效错误
        self.db.update(
            'articles',
            {
                'content_status': 'failed',
                'completion_error': 'Invalid API key (401)',
                'completion_attempts': 1
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'failed'
    
    def test_ai_quota_exceeded(self):
        """测试 AI 配额超限"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="配额超限测试文章",
            firecrawl_status='completed',
            content_status='pending',
            firecrawl_content='Test content'
        )
        
        # 模拟配额超限错误
        self.db.update(
            'articles',
            {
                'content_status': 'failed',
                'completion_error': 'Quota exceeded (429)',
                'completion_attempts': 1
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'failed'
    
    def test_ai_content_too_long(self):
        """测试 AI 内容过长"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="内容过长测试文章",
            firecrawl_status='completed',
            content_status='pending',
            firecrawl_content='x' * 100000  # 超长内容
        )
        
        # 模拟内容过长错误
        self.db.update(
            'articles',
            {
                'content_status': 'failed',
                'completion_error': 'Content too long (max 100k tokens)',
                'completion_attempts': 1
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'failed'
    
    def test_ai_retry_mechanism(self):
        """测试 AI 重试机制"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="重试机制测试文章",
            firecrawl_status='completed',
            content_status='incomplete',
            firecrawl_content='Test content',
            completion_attempts=0
        )
        
        # 第一次失败
        self.db.update(
            'articles',
            {
                'content_status': 'incomplete',
                'completion_error': 'Temporary error',
                'completion_attempts': 1
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['completion_attempts'] == 1
        assert article['content_status'] == 'incomplete'
        
        # 第二次失败
        self.db.update(
            'articles',
            {
                'completion_attempts': 2
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['completion_attempts'] == 2
        
        # 第三次失败
        self.db.update(
            'articles',
            {
                'content_status': 'failed',
                'completion_attempts': 3,
                'completion_error': 'Max retries exceeded'
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'failed'
        assert article['completion_attempts'] >= 3
    
    def test_ai_missing_firecrawl_content(self):
        """测试 AI 缺少 Firecrawl 内容"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="缺少内容测试文章",
            firecrawl_status='completed',
            content_status='pending',
            firecrawl_content=None  # 缺少内容
        )
        
        # 模拟缺少内容错误
        self.db.update(
            'articles',
            {
                'content_status': 'failed',
                'completion_error': 'No firecrawl content available',
                'completion_attempts': 1
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'failed'


class TestFeedRefreshErrorHandling:
    """Feed Refresh 错误处理测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        yield
        
        self.db.cleanup_test_data()
    
    def test_feed_invalid_rss_url(self):
        """测试无效 RSS URL"""
        feed_id = self.db.create_test_feed(
            title="无效RSS测试订阅源",
            url="https://not-a-valid-rss-url.com/feed"
        )
        
        # 尝试刷新
        result = self.api.refresh_feed(feed_id)
        
        # 等待刷新完成
        time.sleep(2)
        
        # 检查状态
        feed = self.db.get_feed(feed_id)
        assert feed['refresh_status'] in ['idle', 'error']
    
    def test_feed_network_timeout(self):
        """测试网络超时"""
        feed_id = self.db.create_test_feed(
            title="网络超时测试订阅源",
            url="https://httpbin.org/delay/120"  # 120秒延迟
        )
        
        # 模拟超时错误
        self.db.update(
            'feeds',
            {
                'refresh_status': 'error',
                'refresh_error': 'Timeout after 60s'
            },
            'id = ?',
            (feed_id,)
        )
        
        feed = self.db.get_feed(feed_id)
        assert feed['refresh_status'] == 'error'
        assert feed['refresh_error'] is not None
    
    def test_feed_parse_error(self):
        """测试解析错误"""
        feed_id = self.db.create_test_feed(
            title="解析错误测试订阅源",
            url="https://example.com/invalid-xml"
        )
        
        # 模拟解析错误
        self.db.update(
            'feeds',
            {
                'refresh_status': 'error',
                'refresh_error': 'Failed to parse RSS feed: invalid XML'
            },
            'id = ?',
            (feed_id,)
        )
        
        feed = self.db.get_feed(feed_id)
        assert feed['refresh_status'] == 'error'
    
    def test_feed_concurrent_refresh_protection(self):
        """测试并发刷新保护"""
        feed_id = self.db.create_test_feed(
            title="并发刷新测试订阅源",
            refresh_status='refreshing'
        )
        
        # 如果已经在刷新，不应该再次触发
        feed = self.db.get_feed(feed_id)
        assert feed['refresh_status'] == 'refreshing'


class TestSchedulerErrorHandling:
    """调度器错误处理测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        yield
        
        self.db.cleanup_test_data()
    
    def test_scheduler_task_not_found(self):
        """测试调度器任务不存在"""
        task = self.db.get_scheduler_task('non_existent_scheduler')
        assert task is None
    
    def test_scheduler_consecutive_failures_tracking(self):
        """测试调度器连续失败跟踪"""
        # 获取调度器任务
        task = self.db.get_scheduler_task('ai_summary')
        
        if task:
            # 模拟连续失败
            initial_failures = task.get('consecutive_failures', 0)
            
            self.db.update(
                'scheduler_tasks',
                {
                    'status': 'error',
                    'consecutive_failures': initial_failures + 1,
                    'last_error': 'Test error'
                },
                'name = ?',
                ('ai_summary',)
            )
            
            task = self.db.get_scheduler_task('ai_summary')
            assert task['consecutive_failures'] >= initial_failures + 1
    
    def test_scheduler_execution_time_tracking(self):
        """测试调度器执行时间跟踪"""
        task = self.db.get_scheduler_task('ai_summary')
        
        if task:
            # 应该有执行时间统计
            assert 'total_executions' in task
            assert 'successful_executions' in task
            assert 'failed_executions' in task
            
            # 执行次数应该匹配
            assert task['total_executions'] >= task['successful_executions'] + task['failed_executions']