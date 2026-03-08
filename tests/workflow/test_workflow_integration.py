"""完整工作流集成测试"""

import pytest
import time
from datetime import datetime
from typing import Dict, Any

from config import TestConfig, DatabaseConfig
from utils import DatabaseHelper, APIClient, MockFirecrawl, MockAIService


class TestFullWorkflow:
    """完整工作流测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        self.mock_firecrawl = MockFirecrawl()
        self.mock_ai = MockAIService()
        
        # 创建测试分类
        self.test_category_id = self.db.insert('categories', {
            **DatabaseConfig.TEST_CATEGORY,
            'created_at': datetime.now().isoformat()
        })
        
        # 创建测试订阅源
        self.test_feed_id = self.db.create_test_feed(
            title="完整流程测试订阅源",
            url="https://example.com/test-workflow-feed",
            firecrawl_enabled=1,
            content_completion_enabled=1,
            max_completion_retries=3
        )
        
        yield
        
        # 清理测试数据
        self.db.cleanup_test_data()
    
    def test_workflow_feed_refresh_to_article_creation(self):
        """测试工作流：Feed Refresh → 创建文章"""
        # 步骤 1: 刷新订阅源
        result = self.api.refresh_feed(self.test_feed_id)
        assert result.get('success'), "刷新订阅源失败"
        
        # 等待刷新完成
        time.sleep(2)
        
        # 步骤 2: 验证文章创建
        articles = self.db.query(
            "SELECT * FROM articles WHERE feed_id = ? ORDER BY created_at DESC LIMIT 5",
            (self.test_feed_id,)
        )
        
        # 如果有新文章，验证状态
        for article in articles:
            if article['firecrawl_status'] == 'pending':
                assert article['content'] is not None, "文章应该有原始内容"
                assert article['content_status'] in ['incomplete', None], "content_status 应该是 incomplete 或 NULL"
    
    def test_workflow_firecrawl_to_ai_summary(self):
        """测试工作流：Firecrawl → AI Summary"""
        # 创建测试文章
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="工作流测试文章",
            link="https://example.com/workflow-test",
            firecrawl_status='pending'
        )
        
        # 步骤 1: 模拟 Firecrawl 完成
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'completed',
                'firecrawl_content': '# Test Article\n\nComplete content from Firecrawl.',
                'content_status': 'incomplete'
            },
            'id = ?',
            (article_id,)
        )
        
        # 验证 Firecrawl 完成
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'completed'
        assert article['content_status'] == 'incomplete'
        
        # 步骤 2: 查询待 AI 总结的文章
        pending_articles = self.db.get_incomplete_ai_summary_articles(limit=10)
        article_ids = [a['id'] for a in pending_articles]
        
        assert article_id in article_ids, "文章应该在待 AI 总结列表中"
        
        # 步骤 3: 模拟 AI 总结完成
        now = datetime.now().isoformat()
        self.db.update(
            'articles',
            {
                'ai_content_summary': '## AI 总结\n\n这是一篇测试文章的 AI 总结。',
                'content_status': 'complete',
                'content_fetched_at': now
            },
            'id = ?',
            (article_id,)
        )
        
        # 验证 AI 总结完成
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'complete'
        assert article['ai_content_summary'] is not None
        assert len(article['ai_content_summary']) > 0
    
    def test_workflow_status_transitions(self):
        """测试工作流：完整状态流转"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="状态流转测试文章",
            firecrawl_status='pending'
        )
        
        # 初始状态
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'pending'
        
        # Firecrawl 处理中
        self.db.update(
            'articles',
            {'firecrawl_status': 'processing'},
            'id = ?',
            (article_id,)
        )
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'processing'
        
        # Firecrawl 完成
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'completed',
                'firecrawl_content': 'Test content',
                'content_status': 'incomplete'
            },
            'id = ?',
            (article_id,)
        )
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'completed'
        assert article['content_status'] == 'incomplete'
        
        # AI 总结处理中
        self.db.update(
            'articles',
            {
                'content_status': 'pending',
                'completion_attempts': 1
            },
            'id = ?',
            (article_id,)
        )
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'pending'
        
        # AI 总结完成
        self.db.update(
            'articles',
            {
                'content_status': 'complete',
                'ai_content_summary': 'Test summary'
            },
            'id = ?',
            (article_id,)
        )
        article = self.db.get_article(article_id)
        assert article['content_status'] == 'complete'
    
    def test_workflow_failure_recovery(self):
        """测试工作流：失败恢复"""
        article_id = self.db.create_test_article(
            feed_id=self.test_feed_id,
            title="失败恢复测试文章",
            firecrawl_status='pending'
        )
        
        # 场景 1: Firecrawl 失败
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'failed',
                'firecrawl_error': 'Network timeout'
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'failed'
        assert article['firecrawl_error'] is not None
        
        # 场景 2: 重试成功
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'completed',
                'firecrawl_content': 'Recovered content',
                'content_status': 'incomplete',
                'firecrawl_error': None
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['firecrawl_status'] == 'completed'
        assert article['content_status'] == 'incomplete'
    
    def test_workflow_with_disabled_firecrawl(self):
        """测试工作流：Firecrawl 禁用"""
        # 创建禁用 Firecrawl 的订阅源
        disabled_feed_id = self.db.create_test_feed(
            title="禁用Firecrawl测试订阅源",
            firecrawl_enabled=0,
            content_completion_enabled=1
        )
        
        article_id = self.db.create_test_article(
            feed_id=disabled_feed_id,
            title="禁用Firecrawl测试文章",
            firecrawl_status='pending'
        )
        
        # 查询待抓取文章
        pending_articles = self.db.get_pending_firecrawl_articles(limit=10)
        article_ids = [a['id'] for a in pending_articles]
        
        # 不应该包含禁用订阅源的文章
        assert article_id not in article_ids, "禁用订阅源的文章不应该在待抓取列表中"
    
    def test_workflow_with_disabled_ai_summary(self):
        """测试工作流：AI Summary 禁用"""
        # 创建禁用 AI Summary 的订阅源
        disabled_feed_id = self.db.create_test_feed(
            title="禁用AI Summary测试订阅源",
            firecrawl_enabled=1,
            content_completion_enabled=0
        )
        
        article_id = self.db.create_test_article(
            feed_id=disabled_feed_id,
            title="禁用AI Summary测试文章",
            firecrawl_status='completed',
            content_status='incomplete',
            firecrawl_content='Test content'
        )
        
        # 查询待 AI 总结文章
        pending_articles = self.db.get_incomplete_ai_summary_articles(limit=10)
        article_ids = [a['id'] for a in pending_articles]
        
        # 不应该包含禁用订阅源的文章
        assert article_id not in article_ids, "禁用订阅源的文章不应该在待总结列表中"
    
    def test_workflow_batch_processing(self):
        """测试工作流：批量处理"""
        # 创建多个测试文章
        article_ids = []
        for i in range(5):
            article_id = self.db.create_test_article(
                feed_id=self.test_feed_id,
                title=f"批量测试文章{i}",
                firecrawl_status='completed',
                content_status='incomplete',
                firecrawl_content=f'Test content {i}'
            )
            article_ids.append(article_id)
        
        # 查询待处理文章
        pending_articles = self.db.get_incomplete_ai_summary_articles(limit=50)
        pending_ids = [a['id'] for a in pending_articles]
        
        # 所有测试文章都应该在待处理列表中
        for article_id in article_ids:
            assert article_id in pending_ids, f"文章 {article_id} 应该在待处理列表中"


class TestWorkflowDataConsistency:
    """工作流数据一致性测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        yield
        
        # 清理测试数据
        self.db.cleanup_test_data()
    
    def test_foreign_key_constraint_feed_to_articles(self):
        """测试外键约束：Feed → Articles"""
        # 创建测试订阅源
        feed_id = self.db.create_test_feed(title="外键测试订阅源")
        
        # 创建测试文章
        article_id = self.db.create_test_article(
            feed_id=feed_id,
            title="外键测试文章"
        )
        
        # 验证关联
        article = self.db.get_article(article_id)
        assert article['feed_id'] == feed_id
        
        # 删除订阅源应该级联删除文章
        self.db.delete('feeds', 'id = ?', (feed_id,))
        
        # 文章应该被删除
        article = self.db.get_article(article_id)
        assert article is None, "删除订阅源后，文章应该被级联删除"
    
    def test_status_field_valid_values(self):
        """测试状态字段有效值"""
        feed_id = self.db.create_test_feed(title="状态字段测试订阅源")
        article_id = self.db.create_test_article(feed_id=feed_id)
        
        # 测试 firecrawl_status
        for status in TestConfig.FIRECRAWL_STATUSES:
            self.db.update(
                'articles',
                {'firecrawl_status': status},
                'id = ?',
                (article_id,)
            )
            article = self.db.get_article(article_id)
            assert article['firecrawl_status'] == status
        
        # 测试 content_status
        for status in TestConfig.CONTENT_STATUSES:
            self.db.update(
                'articles',
                {'content_status': status},
                'id = ?',
                (article_id,)
            )
            article = self.db.get_article(article_id)
            assert article['content_status'] == status
    
    def test_timestamp_fields_update(self):
        """测试时间戳字段更新"""
        feed_id = self.db.create_test_feed(title="时间戳测试订阅源")
        article_id = self.db.create_test_article(feed_id=feed_id)
        
        before_update = datetime.now()
        
        # 更新 Firecrawl 完成时间
        self.db.update(
            'articles',
            {
                'firecrawl_status': 'completed',
                'firecrawl_crawled_at': before_update.isoformat()
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['firecrawl_crawled_at'] is not None
        
        # 更新 AI 总结完成时间
        now = datetime.now().isoformat()
        self.db.update(
            'articles',
            {
                'content_status': 'complete',
                'content_fetched_at': now
            },
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        assert article['content_fetched_at'] is not None
    
    def test_content_length_limits(self):
        """测试内容长度限制"""
        feed_id = self.db.create_test_feed(title="内容长度测试订阅源")
        article_id = self.db.create_test_article(feed_id=feed_id)
        
        # 测试超长内容
        long_content = "x" * (TestConfig.MAX_CONTENT_LENGTH + 1000)
        
        self.db.update(
            'articles',
            {'firecrawl_content': long_content},
            'id = ?',
            (article_id,)
        )
        
        article = self.db.get_article(article_id)
        # 数据库应该能存储超长内容（TEXT 类型）
        assert len(article['firecrawl_content']) > TestConfig.MAX_CONTENT_LENGTH