"""Database helpers for workflow tests."""

import sqlite3
from contextlib import contextmanager
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple


class DatabaseHelper:
    """Helper for direct SQLite access in integration tests."""

    def __init__(self, db_path: str):
        self.db_path = Path(db_path)
        if not self.db_path.exists():
            raise FileNotFoundError(f"Database file not found: {db_path}")

    @contextmanager
    def get_connection(self):
        conn = sqlite3.connect(str(self.db_path))
        conn.row_factory = sqlite3.Row
        conn.execute("PRAGMA foreign_keys = ON")
        try:
            yield conn
        finally:
            conn.close()

    def query(self, sql: str, params: Optional[Tuple] = None) -> List[Dict[str, Any]]:
        with self.get_connection() as conn:
            cursor = conn.cursor()
            if params:
                cursor.execute(sql, params)
            else:
                cursor.execute(sql)
            return [dict(row) for row in cursor.fetchall()]

    def query_one(self, sql: str, params: Optional[Tuple] = None) -> Optional[Dict[str, Any]]:
        results = self.query(sql, params)
        return results[0] if results else None

    def execute(self, sql: str, params: Optional[Tuple] = None) -> int:
        with self.get_connection() as conn:
            cursor = conn.cursor()
            if params:
                cursor.execute(sql, params)
            else:
                cursor.execute(sql)
            conn.commit()
            return cursor.rowcount

    def insert(self, table: str, data: Dict[str, Any]) -> int:
        columns = ', '.join(data.keys())
        placeholders = ', '.join(['?' for _ in data])
        sql = f"INSERT INTO {table} ({columns}) VALUES ({placeholders})"

        with self.get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute(sql, tuple(data.values()))
            conn.commit()
            return cursor.lastrowid

    def update(self, table: str, data: Dict[str, Any], where: str, where_params: Tuple) -> int:
        set_clause = ', '.join([f"{key} = ?" for key in data.keys()])
        sql = f"UPDATE {table} SET {set_clause} WHERE {where}"

        with self.get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute(sql, tuple(data.values()) + where_params)
            conn.commit()
            return cursor.rowcount

    def delete(self, table: str, where: str, where_params: Tuple) -> int:
        with self.get_connection() as conn:
            cursor = conn.cursor()

            if table == 'feeds' and where == 'id = ?':
                feed_id = where_params[0]
                cursor.execute("DELETE FROM ai_summaries WHERE feed_id = ?", (feed_id,))
                cursor.execute("DELETE FROM articles WHERE feed_id = ?", (feed_id,))

            sql = f"DELETE FROM {table} WHERE {where}"
            cursor.execute(sql, where_params)
            conn.commit()
            return cursor.rowcount

    def count(self, table: str, where: Optional[str] = None, where_params: Optional[Tuple] = None) -> int:
        sql = f"SELECT COUNT(*) as count FROM {table}"
        if where:
            sql += f" WHERE {where}"
        result = self.query_one(sql, where_params)
        return result['count'] if result else 0

    def get_article(self, article_id: int) -> Optional[Dict[str, Any]]:
        return self.query_one("SELECT * FROM articles WHERE id = ?", (article_id,))

    def get_feed(self, feed_id: int) -> Optional[Dict[str, Any]]:
        return self.query_one("SELECT * FROM feeds WHERE id = ?", (feed_id,))

    def get_scheduler_task(self, name: str) -> Optional[Dict[str, Any]]:
        return self.query_one("SELECT * FROM scheduler_tasks WHERE name = ?", (name,))

    def get_pending_firecrawl_articles(self, limit: int = 50) -> List[Dict[str, Any]]:
        return self.query(
            """
            SELECT articles.*, feeds.firecrawl_enabled
            FROM articles
            JOIN feeds ON feeds.id = articles.feed_id
            WHERE feeds.firecrawl_enabled = 1
              AND articles.firecrawl_status = 'pending'
            LIMIT ?
            """,
            (limit,),
        )

    def get_incomplete_ai_summary_articles(self, limit: int = 50) -> List[Dict[str, Any]]:
        return self.query(
            """
            SELECT articles.*, feeds.content_completion_enabled
            FROM articles
            JOIN feeds ON feeds.id = articles.feed_id
            WHERE articles.firecrawl_status = 'completed'
              AND articles.content_status = 'incomplete'
              AND feeds.content_completion_enabled = 1
            LIMIT ?
            """,
            (limit,),
        )

    def create_test_article(self, feed_id: int, **kwargs) -> int:
        from datetime import datetime

        data = {
            'feed_id': feed_id,
            'title': kwargs.get('title', 'test article'),
            'link': kwargs.get('link', f"https://example.com/test-{datetime.now().timestamp()}"),
            'description': kwargs.get('description', 'test description'),
            'content': kwargs.get('content', 'test content'),
            'firecrawl_status': kwargs.get('firecrawl_status', 'pending'),
            'content_status': kwargs.get('content_status', None),
            'firecrawl_content': kwargs.get('firecrawl_content', None),
            'firecrawl_error': kwargs.get('firecrawl_error', None),
            'ai_content_summary': kwargs.get('ai_content_summary', None),
            'completion_attempts': kwargs.get('completion_attempts', 0),
            'created_at': datetime.now().isoformat(),
        }
        return self.insert('articles', data)

    def create_test_feed(self, **kwargs) -> int:
        from datetime import datetime

        url = kwargs.get('url', f"https://example.com/feed-{datetime.now().timestamp()}")
        self.execute("DELETE FROM ai_summaries WHERE feed_id IN (SELECT id FROM feeds WHERE url = ?)", (url,))
        self.execute("DELETE FROM articles WHERE feed_id IN (SELECT id FROM feeds WHERE url = ?)", (url,))
        self.execute("DELETE FROM feeds WHERE url = ?", (url,))

        data = {
            'title': kwargs.get('title', 'test feed'),
            'url': url,
            'firecrawl_enabled': kwargs.get('firecrawl_enabled', 1),
            'content_completion_enabled': kwargs.get('content_completion_enabled', 1),
            'max_completion_retries': kwargs.get('max_completion_retries', 3),
            'refresh_interval': kwargs.get('refresh_interval', 60),
            'refresh_status': kwargs.get('refresh_status', 'idle'),
            'created_at': datetime.now().isoformat(),
        }
        return self.insert('feeds', data)

    def cleanup_test_data(self):
        with self.get_connection() as conn:
            cursor = conn.cursor()
            conn.execute("PRAGMA foreign_keys = OFF")
            try:
                cursor.execute(
                    "DELETE FROM ai_summaries WHERE feed_id IN (SELECT id FROM feeds WHERE url LIKE 'https://example.com/%' OR url LIKE 'https://sspai.com/%')"
                )
                cursor.execute(
                    "DELETE FROM ai_summaries WHERE category_id IN (SELECT id FROM categories WHERE slug = 'test-category' OR slug LIKE 'test-%')"
                )
                cursor.execute(
                    "DELETE FROM articles WHERE link LIKE 'https://example.com/%' OR link LIKE 'https://sspai.com/%' OR feed_id IN (SELECT id FROM feeds WHERE url LIKE 'https://example.com/%' OR url LIKE 'https://sspai.com/%')"
                )
                cursor.execute("DELETE FROM feeds WHERE url LIKE 'https://example.com/%' OR url LIKE 'https://sspai.com/%'")
                cursor.execute("DELETE FROM categories WHERE slug = 'test-category' OR slug LIKE 'test-%'")
                conn.commit()
            finally:
                conn.execute("PRAGMA foreign_keys = ON")
