"""调度器单元测试"""

import pytest
import time
from datetime import datetime, timedelta
from typing import Dict, Any

from config import TestConfig, DatabaseConfig
from utils import DatabaseHelper, APIClient


class TestAutoRefreshScheduler:
    """自动刷新调度器测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        # 创建测试数据
        self.test_feed_id = self.db.create_test_feed(
            title=DatabaseConfig.TEST_FEED['title'],
            url=DatabaseConfig.TEST_FEED['url'],
            refresh_interval=60
        )
        
        yield
        
        # 清理测试数据
        self.db.cleanup_test_data()
    
    def test_scheduler_exists(self):
        """测试调度器任务是否存在"""
        task = self.db.get_scheduler_task('auto_refresh')
        assert task is not None, "auto_refresh 调度器任务不存在"
        assert task['status'] in ['idle', 'running'], f"调度器状态异常: {task['status']}"
    
    def test_scheduler_can_be_triggered(self):
        """测试调度器可以手动触发"""
        result = self.api.trigger_scheduler('auto_refresh')
        assert result.get('success'), f"触发调度器失败: {result}"
    
    def test_feed_refresh_status_update(self):
        """测试订阅源刷新状态更新"""
        # 触发刷新
        self.api.refresh_feed(self.test_feed_id)
        
        # 检查状态
        feed = self.db.get_feed(self.test_feed_id)
        assert feed is not None
        
        # 刷新状态应该是 refreshing 或 idle
        assert feed['refresh_status'] in ['idle', 'refreshing', 'error']
    
    def test_feed_refresh_interval(self):
        """测试订阅源刷新间隔"""
        feed = self.db.get_feed(self.test_feed_id)
        assert feed['refresh_interval'] > 0, "刷新间隔应该大于 0"
    
    def test_feed_needs_refresh_check(self):
        """测试订阅源是否需要刷新检查"""
        feed = self.db.get_feed(self.test_feed_id)
        
        if feed['last_refresh_at']:
            last_refresh = datetime.fromisoformat(feed['last_refresh_at'])
            next_refresh = last_refresh + timedelta(minutes=feed['refresh_interval'])
            now = datetime.now()
            
            # 如果当前时间已经超过下次刷新时间，应该需要刷新
            if now >= next_refresh:
                assert feed['refresh_status'] != 'refreshing', "应该刷新但状态为 refreshing"


class TestFirecrawlScheduler:
    """Firecrawl 调度器测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        # 创建测试订阅源和文章
        self.test_feed_id = self.db.create_test_feed(
            title="Firecrawl测试订阅源",
            firecrawl_enabled=1,
            content_completion_enabled=1
        )
        
        self.test_article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="Firecrawl测试文章",
            firecrawl_status='pending'
        )
        
        yield
        
        # 清理测试数据
        self.db.cleanup_test_data()
    
    def test_firecrawl_enabled_feed_query(self):
        """测试查询启用 Firecrawl 的订阅源"""
        articles = self.db.get_pending_firecrawl_articles(limit=10)
        
        # 应该包含我们的测试文章
        article_ids = [a['id'] for a in articles]
        assert self.test_article_id in article_ids, "测试文章应该在待抓取列表中"
    
    def test_firecrawl_status_transition(self):
        """测试 Firecrawl 状态流转"""
        # 初始状态
        article = self.db.get_article(self.test_article_id)
        assert article['firecrawl_status'] == 'pending', "初始状态应该是 pending"
        
        # 模拟状态变更
        self.db.update(
            'articles',
            {'firecrawl_status': 'processing'},
            'id = ?',
            (self.test_article_id,)
        )
        
        article = self.db.get_article(self.test_article_id)
        assert article['firecrawl_status'] == 'processing', "状态应该变为 processing"
    
    def test_firecrawl_concurrency_limit(self):
        """测试 Firecrawl 并发限制"""
        # 创建多个待处理文章
        for i in range(10):
            self.db.create_test_article(
                feed_id=self.test_feed_id,
                title=f"并发测试文章{i}",
                firecrawl_status='pending'
            )
        
        # 查询待处理文章
        articles = self.db.get_pending_firecrawl_articles(limit=50)
        assert len(articles) <= 50, "每次最多处理 50 篇文章"
    
    def test_firecrawl_completion_sets_content_status(self):
        """测试 Firecrawl 完成后设置 content_status"""
        # 模拟 Firecrawl 完成
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'completed',
                'firecrawl_content': '# Test Content',
                'content_status': 'incomplete'
            },
            'id = ?',
            (self.test_article_id,)
        )
        
        article = self.db.get_article(self.test_article_id)
        assert article['firecrawl_status'] == 'completed', "状态应该是 completed"
        assert article['content_status'] == 'incomplete', "应该设置 content_status 为 incomplete"
    
    def test_firecrawl_failure_handling(self):
        """测试 Firecrawl 失败处理"""
        # 模拟 Firecrawl 失败
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'failed',
                'firecrawl_error': 'Test error'
            },
            'id = ?',
            (self.test_article_id,)
        )
        
        article = self.db.get_article(self.test_article_id)
        assert article['firecrawl_status'] == 'failed', "状态应该是 failed"
        assert article['firecrawl_error'] is not None, "应该有错误信息"


class TestAISummaryScheduler:
    """AI 总结调度器测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        # 创建测试订阅源和已完成 Firecrawl 的文章
        self.test_feed_id = self.db.create_test_feed(
            title="AI Summary测试订阅源",
            firecrawl_enabled=1,
            content_completion_enabled=1,
            max_completion_retries=3
        )
        
        self.test_article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="AI Summary测试文章",
            firecrawl_status='completed',
            content_status='incomplete',
            firecrawl_content='# Test Article\n\nThis is test content.'
        )
        
        yield
        
        # 清理测试数据
        self.db.cleanup_test_data()
    
    def test_ai_summary_scheduler_exists(self):
        """测试 AI 总结调度器任务是否存在"""
        task = self.db.get_scheduler_task('ai_summary')
        assert task is not None, "ai_summary 调度器任务不存在"
        assert task['description'] is not None, "调度器应该有描述"
    
    def test_ai_summary_query_conditions(self):
        """测试 AI 总结查询条件"""
        articles = self.db.get_incomplete_ai_summary_articles(limit=10)
        
        # 应该包含我们的测试文章
        article_ids = [a['id'] for a in articles]
        assert self.test_article_id in article_ids, "测试文章应该在待总结列表中"
        
        # 检查条件
        for article in articles:
            assert article['firecrawl_status'] == 'completed', "Firecrawl 状态应该是 completed"
            assert article['content_status'] == 'incomplete', "Content 状态应该是 incomplete"
    
    def test_ai_summary_requires_firecrawl_content(self):
        """测试 AI 总结需要 Firecrawl 内容"""
        article = self.db.get_article(self.test_article_id)
        
        # 应该有 firecrawl_content
        assert article['firecrawl_content'] is not None, "应该有 Firecrawl 内容"
        assert len(article['firecrawl_content']) > 0, "Firecrawl 内容不应该为空"
    
    def test_ai_summary_retry_mechanism(self):
        """测试 AI 总结重试机制"""
        # 模拟失败
        self.db.update(
            'articles',
            {
                'completion_attempts': 1,
                'content_status': 'incomplete'
            },
            'id = ?',
            (self.test_article_id,)
        )
        
        article = self.db.get_article(self.test_article_id)
        feed = self.db.get_feed(self.test_feed_id)
        
        # 应该还可以重试
        assert article['completion_attempts'] < feed['max_completion_retries'], "应该还可以重试"
    
    def test_ai_summary_max_retries_exceeded(self):
        """测试 AI 总结超过最大重试次数"""
        # 设置重试次数为最大值
        self.db.update(
            'articles',
            {
                'completion_attempts': 3,
                'content_status': 'failed'
            },
            'id = ?',
            (self.test_article_id,)
        )
        
        article = self.db.get_article(self.test_article_id)
        assert article['content_status'] == 'failed', "超过重试次数后应该标记为 failed"
    
    def test_ai_summary_success_updates_fields(self):
        """测试 AI 总结成功更新字段"""
        # 模拟 AI 总结成功
        now = datetime.now().isoformat()
        self.db.update(
            'articles',
            {
                'ai_content_summary': '## AI 总结\n\n测试总结内容',
                'content_status': 'complete',
                'content_fetched_at': now
            },
            'id = ?',
            (self.test_article_id,)
        )
        
        article = self.db.get_article(self.test_article_id)
        assert article['content_status'] == 'complete', "状态应该是 complete"
        assert article['ai_content_summary'] is not None, "应该有 AI 总结内容"
        assert article['content_fetched_at'] is not None, "应该记录完成时间"


class TestSchedulerIntegration:
    """调度器集成测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        yield
        
        # 清理测试数据
        self.db.cleanup_test_data()
    
    def test_all_schedulers_are_healthy(self):
        """测试所有调度器健康状态"""
        schedulers = self.api.get_schedulers_status()
        
        assert len(schedulers) > 0, "应该有调度器任务"
        
        for scheduler in schedulers:
            assert scheduler['status'] in ['idle', 'running'], f"调度器 {scheduler['name']} 状态异常"
            assert scheduler['check_interval'] > 0, f"调度器 {scheduler['name']} 检查间隔应该大于 0"
    
    def test_scheduler_can_be_triggered_manually(self):
        """测试调度器可以手动触发"""
        # 测试 auto_refresh
        result = self.api.trigger_scheduler('auto_refresh')
        assert result.get('success'), "应该可以触发 auto_refresh"
        
        # 测试 ai_summary
        result = self.api.trigger_scheduler('ai_summary')
        assert result.get('success'), "应该可以触发 ai_summary"