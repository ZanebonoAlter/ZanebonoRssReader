"""API 客户端工具"""

import time
from typing import Dict, Any, Optional, List
import requests


class APIClient:
    """后端 API 客户端"""
    
    def __init__(self, base_url: str, timeout: int = 60):
        self.base_url = base_url.rstrip('/')
        self.timeout = timeout
        self.session = requests.Session()
    
    def _request(
        self, 
        method: str, 
        endpoint: str, 
        **kwargs
    ) -> Dict[str, Any]:
        """统一请求方法"""
        url = f"{self.base_url}{endpoint}"
        kwargs.setdefault('timeout', self.timeout)
        
        response = self.session.request(method, url, **kwargs)
        
        if response.status_code >= 400:
            raise Exception(f"HTTP {response.status_code}: {response.text}")
        
        data = response.json()
        
        if not data.get('success'):
            raise Exception(f"API Error: {data.get('error', 'Unknown error')}")
        
        return data
    
    def get(self, endpoint: str, params: Optional[Dict] = None) -> Dict[str, Any]:
        """GET 请求"""
        return self._request('GET', endpoint, params=params)
    
    def post(self, endpoint: str, json: Optional[Dict] = None) -> Dict[str, Any]:
        """POST 请求"""
        return self._request('POST', endpoint, json=json)
    
    def put(self, endpoint: str, json: Optional[Dict] = None) -> Dict[str, Any]:
        """PUT 请求"""
        return self._request('PUT', endpoint, json=json)
    
    def delete(self, endpoint: str) -> Dict[str, Any]:
        """DELETE 请求"""
        return self._request('DELETE', endpoint)
    
    # ============ 分类 API ============
    def get_categories(self) -> List[Dict[str, Any]]:
        """获取所有分类"""
        result = self.get('/api/categories')
        return result.get('data', [])
    
    def create_category(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """创建分类"""
        return self.post('/api/categories', json=data)
    
    # ============ 订阅源 API ============
    def get_feeds(self) -> List[Dict[str, Any]]:
        """获取所有订阅源"""
        result = self.get('/api/feeds')
        return result.get('data', [])
    
    def get_feed(self, feed_id: int) -> Dict[str, Any]:
        """获取单个订阅源"""
        return self.get(f'/api/feeds/{feed_id}')
    
    def create_feed(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """创建订阅源"""
        return self.post('/api/feeds', json=data)
    
    def update_feed(self, feed_id: int, data: Dict[str, Any]) -> Dict[str, Any]:
        """更新订阅源"""
        return self.put(f'/api/feeds/{feed_id}', json=data)
    
    def delete_feed(self, feed_id: int) -> Dict[str, Any]:
        """删除订阅源"""
        return self.delete(f'/api/feeds/{feed_id}')
    
    def refresh_feed(self, feed_id: int) -> Dict[str, Any]:
        """刷新订阅源"""
        return self.post(f'/api/feeds/{feed_id}/refresh')
    
    # ============ 文章 API ============
    def get_articles(self, feed_id: Optional[int] = None, limit: int = 50) -> List[Dict[str, Any]]:
        """获取文章列表"""
        params = {'limit': limit}
        if feed_id:
            params['feed_id'] = feed_id
        
        result = self.get('/api/articles', params=params)
        return result.get('data', [])
    
    def get_article(self, article_id: int) -> Dict[str, Any]:
        """获取单个文章"""
        return self.get(f'/api/articles/{article_id}')
    
    def update_article(self, article_id: int, data: Dict[str, Any]) -> Dict[str, Any]:
        """更新文章"""
        return self.put(f'/api/articles/{article_id}', json=data)
    
    def fetch_article_content(self, article_id: int) -> Dict[str, Any]:
        """抓取文章内容"""
        return self.post(f'/api/articles/{article_id}/content')
    
    # ============ 调度器 API ============
    def get_schedulers_status(self) -> List[Dict[str, Any]]:
        """获取所有调度器状态"""
        result = self.get('/api/schedulers/status')
        return result.get('data', [])
    
    def get_scheduler_status(self, name: str) -> Dict[str, Any]:
        """获取特定调度器状态"""
        return self.get(f'/api/schedulers/{name}/status')
    
    def trigger_scheduler(self, name: str) -> Dict[str, Any]:
        """手动触发调度器"""
        return self.post(f'/api/schedulers/{name}/trigger')
    
    # ============ Firecrawl API ============
    def get_firecrawl_status(self) -> Dict[str, Any]:
        """获取 Firecrawl 状态"""
        return self.get('/api/firecrawl/status')
    
    def enable_feed_firecrawl(self, feed_id: int, enabled: bool = True) -> Dict[str, Any]:
        """启用/禁用订阅源的 Firecrawl"""
        return self.post(f'/api/firecrawl/feed/{feed_id}/enable', json={'enabled': enabled})
    
    # ============ AI 设置 API ============
    def get_ai_settings(self) -> Dict[str, Any]:
        """获取 AI 设置"""
        return self.get('/api/ai/settings')
    
    def update_ai_settings(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """更新 AI 设置"""
        return self.post('/api/ai/settings', json=data)
    
    # ============ 健康检查 ============
    def health_check(self) -> bool:
        """健康检查"""
        try:
            self.get('/api/categories')
            return True
        except Exception:
            return False
    
    def wait_for_service(self, max_retries: int = 10, delay: int = 2) -> bool:
        """等待服务就绪"""
        for i in range(max_retries):
            if self.health_check():
                return True
            time.sleep(delay)
        return False