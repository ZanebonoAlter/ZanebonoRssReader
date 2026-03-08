"""模拟服务工具"""

from typing import Dict, Any, Optional
from unittest.mock import Mock, patch


class MockFirecrawl:
    """模拟 Firecrawl 服务"""
    
    def __init__(self, base_url: str = "http://localhost:3002"):
        self.base_url = base_url
        self.enabled = True
        self.call_count = 0
        self.fail_next = False
    
    def mock_scrape_response(
        self,
        markdown: str = "# Test Content\n\nThis is test content.",
        html: str = "<h1>Test Content</h1><p>This is test content.</p>",
        success: bool = True,
        error: Optional[str] = None
    ) -> Dict[str, Any]:
        """生成模拟的抓取响应"""
        self.call_count += 1
        
        if self.fail_next:
            self.fail_next = False
            return {
                "success": False,
                "error": error or "Mocked failure"
            }
        
        if not success:
            return {
                "success": False,
                "error": error or "Scrape failed"
            }
        
        return {
            "success": True,
            "data": {
                "markdown": markdown,
                "html": html,
                "metadata": {
                    "title": "Test Article",
                    "description": "Test description",
                    "language": "zh"
                }
            }
        }
    
    def set_fail_next(self, fail: bool = True):
        """设置下一次调用失败"""
        self.fail_next = fail
    
    def reset(self):
        """重置状态"""
        self.call_count = 0
        self.fail_next = False


class MockAIService:
    """模拟 AI 服务"""
    
    def __init__(self):
        self.enabled = True
        self.configured = True
        self.call_count = 0
        self.fail_next = False
        self.delay_seconds = 0
    
    def mock_summarize_response(
        self,
        one_sentence: str = "这是一篇测试文章。",
        key_points: Optional[list] = None,
        takeaways: Optional[list] = None,
        tags: Optional[list] = None,
        success: bool = True,
        error: Optional[str] = None
    ) -> Dict[str, Any]:
        """生成模拟的 AI 总结响应"""
        import time
        
        self.call_count += 1
        
        if self.delay_seconds > 0:
            time.sleep(self.delay_seconds)
        
        if self.fail_next:
            self.fail_next = False
            return {
                "success": False,
                "error": error or "Mocked AI failure"
            }
        
        if not success:
            return {
                "success": False,
                "error": error or "AI summarization failed"
            }
        
        if key_points is None:
            key_points = ["要点1", "要点2", "要点3"]
        
        if takeaways is None:
            takeaways = ["收获1", "收获2"]
        
        if tags is None:
            tags = ["测试", "示例"]
        
        return {
            "success": True,
            "data": {
                "one_sentence": one_sentence,
                "key_points": key_points,
                "takeaways": takeaways,
                "tags": tags
            }
        }
    
    def set_fail_next(self, fail: bool = True):
        """设置下一次调用失败"""
        self.fail_next = fail
    
    def set_delay(self, seconds: int):
        """设置延迟"""
        self.delay_seconds = seconds
    
    def reset(self):
        """重置状态"""
        self.call_count = 0
        self.fail_next = False
        self.delay_seconds = 0


class MockRSSParser:
    """模拟 RSS 解析器"""
    
    def __init__(self):
        self.articles = []
    
    def add_article(
        self,
        title: str = "测试文章",
        link: str = "https://example.com/article",
        description: str = "测试描述",
        content: str = "测试内容",
        author: str = "测试作者"
    ):
        """添加测试文章"""
        self.articles.append({
            "title": title,
            "link": link,
            "description": description,
            "content": content,
            "author": author
        })
    
    def get_articles(self):
        """获取文章列表"""
        return self.articles.copy()
    
    def clear(self):
        """清空文章列表"""
        self.articles.clear()