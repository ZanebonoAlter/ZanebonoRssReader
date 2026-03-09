#!/usr/bin/env python3
"""Firecrawl完整集成测试脚本"""

import json
import sqlite3
import time
from pathlib import Path
from typing import Dict, Any, Optional, List

import requests

from config import TestConfig


class TestReport:
    """测试报告生成器"""
    
    def __init__(self):
        self.steps: List[Dict[str, Any]] = []
        self.start_time: Optional[float] = None
        self.end_time: Optional[float] = None
        
    def start(self):
        """开始测试"""
        self.start_time = time.time()
        
    def add_step(self, step_name: str, success: bool, duration: float, details: str):
        """记录测试步骤结果"""
        self.steps.append({
            "step": step_name,
            "success": success,
            "duration": duration,
            "details": details
        })
        
    def finish(self):
        """结束测试"""
        self.end_time = time.time()
        
    def generate_report(self) -> str:
        """生成测试报告"""
        report = []
        report.append("=" * 70)
        report.append("Firecrawl集成测试报告")
        report.append("=" * 70)
        
        if self.start_time and self.end_time:
            total_duration = self.end_time - self.start_time
            report.append(f"\n总耗时: {total_duration:.2f}秒\n")
        
        for idx, step in enumerate(self.steps, 1):
            status = "[PASS]" if step["success"] else "[FAIL]"
            report.append(f"{idx}. {step['step']}: {status}")
            report.append(f"   耗时: {step['duration']:.2f}秒")
            report.append(f"   详情: {step['details']}")
            report.append("")
        
        passed = sum(1 for s in self.steps if s["success"])
        total = len(self.steps)
        report.append("=" * 70)
        report.append(f"测试结果: {passed}/{total} 通过")
        report.append("=" * 70)
        
        return "\n".join(report)


class FirecrawlIntegrationTest:
    """Firecrawl完整集成测试"""
    
    def __init__(self):
        self.config = TestConfig()
        self.report = TestReport()
        
    # ============ 工具方法 ============
    def _http_request(
        self, 
        method: str, 
        url: str, 
        timeout: int = 60,
        **kwargs
    ) -> requests.Response:
        """统一的HTTP请求封装"""
        kwargs.setdefault('timeout', timeout)
        response = requests.request(method, url, **kwargs)
        return response
        
    def _db_query(self, sql: str, params: Optional[tuple] = None) -> List[tuple]:
        """数据库查询封装"""
        db_path = Path(self.config.DATABASE_PATH)
        if not db_path.exists():
            raise FileNotFoundError(f"数据库文件不存在: {db_path}")
            
        conn = sqlite3.connect(str(db_path))
        cursor = conn.cursor()
        
        try:
            if params:
                cursor.execute(sql, params)
            else:
                cursor.execute(sql)
            results = cursor.fetchall()
            return results
        finally:
            conn.close()
            
    def _db_execute(self, sql: str, params: Optional[tuple] = None):
        """数据库执行（更新/插入）封装"""
        db_path = Path(self.config.DATABASE_PATH)
        conn = sqlite3.connect(str(db_path))
        cursor = conn.cursor()
        
        try:
            if params:
                cursor.execute(sql, params)
            else:
                cursor.execute(sql)
            conn.commit()
        finally:
            conn.close()
            
    def _verify_response(
        self, 
        response: requests.Response, 
        expected_status: int = 200
    ) -> Dict[str, Any]:
        """响应验证工具"""
        if response.status_code != expected_status:
            raise AssertionError(
                f"HTTP状态码错误: 期望{expected_status}, 实际{response.status_code}"
            )
            
        data = response.json()
        if not data.get("success"):
            error = data.get("error", "未知错误")
            raise AssertionError(f"API返回失败: {error}")
            
        return data
        
    def _execute_test_step(self, step_name: str, test_func) -> bool:
        """测试步骤执行器"""
        start = time.time()
        success = False
        details = ""
        
        try:
            test_func()
            success = True
            details = "测试通过"
        except Exception as e:
            details = f"测试失败: {str(e)}"
            
        duration = time.time() - start
        self.report.add_step(step_name, success, duration, details)
        return success
    
    # ============ 测试步骤 ============
    def test_1_firecrawl_health(self):
        """测试步骤1: Firecrawl服务健康检查"""
        url = f"{self.config.FIRECRAWL_BASE_URL}/health"
        response = self._http_request("GET", url, timeout=10)
        
        if response.status_code != 200:
            raise AssertionError(
                f"Firecrawl服务不可用: HTTP {response.status_code}"
            )
        
        data = response.json()
        if data.get("status") != "ok":
            raise AssertionError(f"Firecrawl服务状态异常: {data}")
    
    def test_2_backend_health(self):
        """测试步骤2: 后端服务健康检查"""
        url = f"{self.config.BACKEND_BASE_URL}/api/categories"
        response = self._http_request("GET", url, timeout=10)
        
        self._verify_response(response)
    
    def test_3_firecrawl_scrape(self):
        """测试步骤3: Firecrawl爬取测试"""
        url = f"{self.config.FIRECRAWL_BASE_URL}/v1/scrape"
        payload = {
            "url": self.config.TEST_ARTICLE_URL
        }
        
        response = self._http_request(
            "POST",
            url,
            json=payload,
            timeout=self.config.FIRECRAWL_TIMEOUT
        )
        
        if response.status_code != 200:
            raise AssertionError(
                f"爬取失败: HTTP {response.status_code}"
            )
        
        data = response.json()
        if not data.get("success"):
            raise AssertionError(f"爬取返回失败: {data}")
        
        content = data.get("data", {}).get("markdown", "")
        if len(content) < 100:
            raise AssertionError(f"爬取内容过短: {len(content)}字符")
    
    def test_4_article_content_update(self):
        """测试步骤4: 文章内容更新测试"""
        # 清空测试文章内容
        self._db_execute(
            "UPDATE articles SET content = NULL WHERE id = ?",
            (self.config.TEST_ARTICLE_ID,)
        )
        
        # 调用后端API更新文章内容
        url = f"{self.config.BACKEND_BASE_URL}/api/articles/{self.config.TEST_ARTICLE_ID}/content"
        response = self._http_request("POST", url, timeout=self.config.BACKEND_TIMEOUT)
        
        data = self._verify_response(response)
        
        # 验证数据库内容已更新
        results = self._db_query(
            "SELECT content FROM articles WHERE id = ?",
            (self.config.TEST_ARTICLE_ID,)
        )
        
        if not results or not results[0][0]:
            raise AssertionError("文章内容未更新到数据库")
        
        content = results[0][0]
        if len(content) < 100:
            raise AssertionError(f"数据库内容过短: {len(content)}字符")
    
    def test_5_ai_settings_config(self):
        """测试步骤5: AI配置验证"""
        results = self._db_query(
            "SELECT value FROM ai_settings WHERE key = 'summary_config'"
        )
        
        if not results:
            raise AssertionError("AI配置不存在")
        
        config_str = results[0][0]
        config = json.loads(config_str)
        
        if "firecrawl" not in config:
            raise AssertionError("Firecrawl配置缺失")
        
        firecrawl_config = config["firecrawl"]
        if not firecrawl_config.get("enabled"):
            raise AssertionError("Firecrawl未启用")
        
        if firecrawl_config.get("api_url") != self.config.FIRECRAWL_BASE_URL:
            raise AssertionError(
                f"Firecrawl API URL不匹配: {firecrawl_config.get('api_url')}"
            )
    
    # ============ 主测试流程 ============
    def run_all_tests(self):
        """运行所有测试"""
        print("\n" + "=" * 70)
        print("开始Firecrawl集成测试")
        print("=" * 70 + "\n")
        
        self.report.start()
        
        # 执行测试步骤
        tests = [
            ("Firecrawl服务健康检查", self.test_1_firecrawl_health),
            ("后端服务健康检查", self.test_2_backend_health),
            ("Firecrawl爬取功能", self.test_3_firecrawl_scrape),
            ("文章内容更新", self.test_4_article_content_update),
            ("AI配置验证", self.test_5_ai_settings_config),
        ]
        
        for step_name, test_func in tests:
            print(f"执行测试: {step_name}...")
            success = self._execute_test_step(step_name, test_func)
            if success:
                print(f"  [PASS] 通过\n")
            else:
                print(f"  [FAIL] 失败\n")
        
        self.report.finish()
        
        # 输出报告
        print(self.report.generate_report())
        
        # 返回是否全部通过
        passed = sum(1 for s in self.report.steps if s["success"])
        total = len(self.report.steps)
        return passed == total


def main():
    """主函数"""
    test = FirecrawlIntegrationTest()
    success = test.run_all_tests()
    
    if success:
        print("\n[PASS] 所有测试通过！")
        exit(0)
    else:
        print("\n[FAIL] 部分测试失败")
        exit(1)


if __name__ == "__main__":
    main()