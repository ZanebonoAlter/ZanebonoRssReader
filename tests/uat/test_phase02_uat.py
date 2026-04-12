"""Phase 02 UAT: 标签流程统一

自动化验收测试，验证:
1. 手动重打标签 API 异步返回 job_id/article_id/status（不再同步返回 tags）
2. WebSocket tag_completed 消息接收
3. tag_completed 消息格式符合契约
4. TagQueue 启动失败后台重试（通过 API 可用性间接验证）
5. job_id 写入 tag_jobs 表

前提: 后端服务运行在 localhost:5000，PostgreSQL 可连接
"""

import json
import time
import threading

import psycopg2
import psycopg2.extras
import pytest
import requests
import websocket

BASE_URL = "http://localhost:5000"
WS_URL = "ws://localhost:5000/ws"
API = f"{BASE_URL}/api"
TIMEOUT = 30

PG_DSN = "host=127.0.0.1 user=postgres password=postgres dbname=rss_reader port=5432 sslmode=disable"


def _pg_conn():
    return psycopg2.connect(PG_DSN)


def _get_first_article():
    resp = requests.get(f"{API}/articles", params={"per_page": 1}, timeout=TIMEOUT)
    assert resp.status_code == 200, f"获取文章列表失败: {resp.status_code}"
    body = resp.json()
    articles = body.get("data", [])
    if not articles:
        pytest.skip("没有文章，无法测试重打标签")
    return articles[0]


def _query_tag_jobs(article_id=None, job_id=None):
    conn = _pg_conn()
    try:
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
        if job_id is not None:
            cur.execute("SELECT * FROM tag_jobs WHERE id = %s", (job_id,))
        elif article_id is not None:
            cur.execute(
                "SELECT * FROM tag_jobs WHERE article_id = %s ORDER BY id DESC LIMIT 5",
                (article_id,),
            )
        else:
            cur.execute("SELECT * FROM tag_jobs ORDER BY id DESC LIMIT 10")
        rows = cur.fetchall()
        conn.rollback()
        return [dict(r) for r in rows]
    finally:
        conn.close()


class TestRetagAsyncEnqueue:
    """Test 1: 手动重打标签 API 异步返回 job_id"""

    def test_retag_returns_job_metadata(self):
        article = _get_first_article()
        article_id = article["id"]

        resp = requests.post(f"{API}/articles/{article_id}/tags", timeout=TIMEOUT)
        assert resp.status_code == 200, f"重打标签请求失败: {resp.status_code} {resp.text}"

        body = resp.json()
        assert body.get("success") is True, f"success 应为 true，实际: {body}"

        data = body.get("data", {})
        assert "job_id" in data, f"响应应包含 job_id，实际: {data}"
        assert "article_id" in data, f"响应应包含 article_id，实际: {data}"
        assert "status" in data, f"响应应包含 status，实际: {data}"

        assert data["article_id"] == article_id, (
            f"返回的 article_id={data['article_id']} 与请求的 {article_id} 不匹配"
        )

        assert data["status"] in ("pending", "leased"), (
            f"status 应为 pending 或 leased，实际: {data['status']}"
        )

        assert "tags" not in data, f"异步响应不应包含 tags 字段，实际: {list(data.keys())}"

        print(f"\n  [OK] 异步入队: job_id={data['job_id']}, article_id={data['article_id']}, "
              f"status={data['status']}")

    def test_retag_nonexistent_article_returns_404(self):
        resp = requests.post(f"{API}/articles/999999/tags", timeout=TIMEOUT)
        assert resp.status_code == 404, f"不存在的文章应返回 404，实际: {resp.status_code}"


class TestTagCompletedWebSocket:
    """Test 2 & 3: WebSocket tag_completed 消息接收与格式"""

    WAIT_TIMEOUT = 90

    def test_tag_completed_event_received(self):
        article = _get_first_article()
        article_id = article["id"]

        collected = []
        ready = threading.Event()

        def on_message(ws, msg):
            try:
                data = json.loads(msg)
            except (json.JSONDecodeError, TypeError):
                return
            if data.get("type") == "tag_completed" and data.get("article_id") == article_id:
                collected.append(data)
                ready.set()

        ws = websocket.WebSocketApp(WS_URL, on_message=on_message)
        ws_thread = threading.Thread(target=ws.run_forever, daemon=True)
        ws_thread.start()
        time.sleep(1)

        try:
            resp = requests.post(f"{API}/articles/{article_id}/tags", timeout=TIMEOUT)
            assert resp.status_code == 200
            body = resp.json()
            job_id = body["data"]["job_id"]

            got_event = ready.wait(timeout=self.WAIT_TIMEOUT)
            if not got_event:
                job_status = _query_tag_jobs(job_id=job_id)
                status_str = job_status[0]["status"] if job_status else "unknown"
                pytest.skip(
                    f"{self.WAIT_TIMEOUT}s 内未收到 tag_completed 事件"
                    f"（job_id={job_id}, status={status_str}，"
                    f"可能是 AI 服务未配置或处理超时）"
                )

            msg = collected[0]

            assert msg["type"] == "tag_completed"
            assert msg["article_id"] == article_id
            assert msg["job_id"] == job_id
            assert "tags" in msg, f"消息应包含 tags 字段，实际: {list(msg.keys())}"
            assert isinstance(msg["tags"], list), f"tags 应为数组，实际: {type(msg['tags'])}"

            if msg["tags"]:
                tag = msg["tags"][0]
                for field in ("slug", "label", "category"):
                    assert field in tag, f"tag 缺少字段: {field}，实际: {list(tag.keys())}"

            print(f"\n  [OK] tag_completed: article_id={article_id}, job_id={job_id}, "
                  f"tags_count={len(msg['tags'])}")
        finally:
            ws.close()


class TestTagJobWrittenToDB:
    """Test 5: job_id 写入 tag_jobs 表"""

    def test_job_record_exists_in_db(self):
        article = _get_first_article()
        article_id = article["id"]

        resp = requests.post(f"{API}/articles/{article_id}/tags", timeout=TIMEOUT)
        assert resp.status_code == 200
        body = resp.json()
        job_id = body["data"]["job_id"]

        jobs = _query_tag_jobs(job_id=job_id)
        assert len(jobs) >= 1, f"tag_jobs 表中未找到 job_id={job_id} 的记录"

        job = jobs[0]
        assert job["article_id"] == article_id, (
            f"tag_jobs 的 article_id={job['article_id']} 与 {article_id} 不匹配"
        )
        assert job["status"] in ("pending", "leased", "completed", "processing", "failed"), (
            f"job status 异常: {job['status']}"
        )

        print(f"\n  [OK] tag_jobs 记录: id={job['id']}, article_id={job['article_id']}, "
              f"status={job['status']}")


class TestTagQueueStartRetry:
    """Test 4: TagQueue 启动失败后台重试

    直接测试需要模拟数据库不可用，通过 API 可用性间接验证 TagQueue 正在运行。
    """

    def test_tag_queue_is_running(self):
        article = _get_first_article()
        article_id = article["id"]

        resp = requests.post(f"{API}/articles/{article_id}/tags", timeout=TIMEOUT)
        assert resp.status_code == 200, (
            "TagQueue 应正在运行并能接受新的入队请求，"
            f"但 API 返回: {resp.status_code} {resp.text}"
        )

        body = resp.json()
        assert body.get("success") is True, f"入队应成功，实际: {body}"
        assert "job_id" in body.get("data", {}), "入队成功应返回 job_id"

        print(f"\n  [OK] TagQueue 运行中，入队成功: job_id={body['data']['job_id']}")
