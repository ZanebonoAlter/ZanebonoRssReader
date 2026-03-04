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
        report.append("📊 Firecrawl集成测试报告")
        report.append("=" * 70)
        
        if self.start_time and self.end_time:
            total_duration = self.end_time - self.start_time
            report.append(f"\n总耗时: {total_duration:.2f}秒\n")
        
        for idx, step in enumerate(self.steps, 1):
            status = "✅" if step["success"] else "❌"
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