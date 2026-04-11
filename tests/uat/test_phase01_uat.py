"""Phase 01 UAT: 并发控制修复

自动化验收测试，验证:
1. TriggerNow 锁冲突返回 HTTP 409 Conflict（同步 scheduler）
2. Firecrawl TriggerNow 返回 batch_id
3. batch_id 与 WebSocket firecrawl_progress 一致
4. Auto-refresh 完成后广播 auto_refresh_complete
5. auto_refresh_complete 消息包含完整字段

前提: 后端服务运行在 localhost:5000
"""

import json
import time
import threading
from datetime import datetime

import pytest
import requests
import websocket

BASE_URL = "http://localhost:5000"
WS_URL = "ws://localhost:5000/ws"
API = f"{BASE_URL}/api/schedulers"
TIMEOUT = 30


def _trigger(name: str) -> requests.Response:
    return requests.post(f"{API}/{name}/trigger", timeout=TIMEOUT)


def _reset(name: str) -> requests.Response:
    return requests.post(f"{API}/{name}/reset", timeout=TIMEOUT)


class TestTriggerNowConflict:
    """Test 1: TriggerNow 锁冲突返回 409

    auto_refresh 是同步执行（阻塞到刷新完成），适合竞态测试。
    其他 scheduler 异步执行且完成太快，第二次 trigger 时锁已释放。
    用并发线程在第一次 trigger 返回前发起第二次请求来验证锁机制。
    """

    @pytest.fixture(autouse=True)
    def reset_schedulers(self):
        _reset("auto_refresh")
        time.sleep(0.5)
        yield
        _reset("auto_refresh")

    def test_auto_refresh_conflict_on_concurrent_trigger(self):
        results = {}

        def trigger_thread(key):
            results[key] = _trigger("auto_refresh")

        t1 = threading.Thread(target=trigger_thread, args=("first",))
        t2 = threading.Thread(target=trigger_thread, args=("second",))

        t1.start()
        time.sleep(0.05)
        t2.start()

        t1.join(timeout=TIMEOUT)
        t2.join(timeout=TIMEOUT)

        statuses = {k: v.status_code for k, v in results.items()}
        bodies = {k: v.json() for k, v in results.items()}

        has_200 = 200 in statuses.values()
        has_409 = 409 in statuses.values()

        assert has_200, f"至少一次应返回 200，实际: {statuses}"
        assert has_409, f"并发触发应有一次返回 409，实际: {statuses}"

        for key, body in bodies.items():
            if body.get("data", {}).get("accepted") is False:
                reason = str(body.get("data", {}).get("reason", "")).lower()
                assert "already_running" in reason, (
                    f"锁冲突的 reason 应包含 already_running，实际: {reason}"
                )

    def test_firecrawl_conflict_on_concurrent_trigger(self):
        _reset("firecrawl")
        time.sleep(0.5)

        results = {}

        def trigger_thread(key):
            results[key] = _trigger("firecrawl")

        t1 = threading.Thread(target=trigger_thread, args=("first",))
        t2 = threading.Thread(target=trigger_thread, args=("second",))

        t1.start()
        time.sleep(0.01)
        t2.start()

        t1.join(timeout=TIMEOUT)
        t2.join(timeout=TIMEOUT)
        _reset("firecrawl")

        statuses = {k: v.status_code for k, v in results.items()}
        has_200 = 200 in statuses.values()
        has_409 = 409 in statuses.values()

        assert has_200, f"至少一次应返回 200，实际: {statuses}"
        assert has_409, f"并发触发应有一次返回 409，实际: {statuses}"

    def test_content_completion_conflict_on_concurrent_trigger(self):
        _reset("content_completion")
        time.sleep(0.5)

        results = {}

        def trigger_thread(key):
            results[key] = _trigger("content_completion")

        t1 = threading.Thread(target=trigger_thread, args=("first",))
        t2 = threading.Thread(target=trigger_thread, args=("second",))

        t1.start()
        time.sleep(0.01)
        t2.start()

        t1.join(timeout=TIMEOUT)
        t2.join(timeout=TIMEOUT)
        _reset("content_completion")

        statuses = {k: v.status_code for k, v in results.items()}
        has_200 = 200 in statuses.values()
        has_409 = 409 in statuses.values()

        assert has_200, f"至少一次应返回 200，实际: {statuses}"
        assert has_409, f"并发触发应有一次返回 409，实际: {statuses}"


class TestFirecrawlBatchId:
    """Test 2: Firecrawl TriggerNow 返回 batch_id"""

    @pytest.fixture(autouse=True)
    def reset_firecrawl(self):
        _reset("firecrawl")
        time.sleep(0.5)
        yield
        _reset("firecrawl")

    def test_batch_id_in_response(self):
        resp = _trigger("firecrawl")
        assert resp.status_code == 200, f"触发应成功，实际: {resp.status_code} {resp.text}"
        body = resp.json()
        data = body.get("data", {})
        batch_id = data.get("batch_id")
        assert batch_id is not None, f"响应中应包含 batch_id，实际 data: {data}"
        assert isinstance(batch_id, str), f"batch_id 应为字符串，实际: {type(batch_id)}"
        assert len(batch_id) > 0, "batch_id 不应为空字符串"

        try:
            datetime.strptime(batch_id, "%Y%m%d%H%M%S")
        except ValueError:
            pytest.fail(f"batch_id 格式应为 YYYYMMDDHHmmss，实际: {batch_id}")


class TestBatchIdWsConsistency:
    """Test 3: batch_id 与 WebSocket 进度一致"""

    @pytest.fixture(autouse=True)
    def reset_firecrawl(self):
        _reset("firecrawl")
        time.sleep(0.5)
        yield
        _reset("firecrawl")

    def test_batch_id_matches_ws(self):
        collected = []
        ready = threading.Event()

        def on_message(ws, msg):
            try:
                data = json.loads(msg)
            except (json.JSONDecodeError, TypeError):
                return
            if data.get("type") == "firecrawl_progress":
                collected.append(data)
                if data.get("status") in ("processing", "completed", "failed"):
                    ready.set()

        ws = websocket.WebSocketApp(WS_URL, on_message=on_message)
        ws_thread = threading.Thread(target=ws.run_forever, daemon=True)
        ws_thread.start()
        time.sleep(1)

        try:
            resp = _trigger("firecrawl")
            assert resp.status_code == 200
            api_batch_id = resp.json()["data"]["batch_id"]

            got_event = ready.wait(timeout=60)
            if not got_event and not collected:
                pytest.skip("60s 内未收到 firecrawl_progress 事件（可能无待抓取文章）")

            ws_batch_ids = {m.get("batch_id") for m in collected}
            assert api_batch_id in ws_batch_ids, (
                f"API batch_id={api_batch_id} 不在 WS batch_ids={ws_batch_ids} 中"
            )
        finally:
            ws.close()


class TestAutoRefreshCompleteBroadcast:
    """Test 4 & 5: Auto-refresh 完成广播

    auto_refresh_complete 只在有 feed 被触发刷新时才广播。
    先检查当前是否有需要刷新的 feed，如果没有则 skip。
    """

    REQUIRED_FIELDS = ["type", "triggered_feeds", "stale_reset_feeds", "duration_seconds", "timestamp"]

    @pytest.fixture(autouse=True)
    def reset_auto_refresh(self):
        _reset("auto_refresh")
        time.sleep(0.5)
        yield
        _reset("auto_refresh")

    def _ensure_feed_needs_refresh(self):
        feeds_resp = requests.get(f"{BASE_URL}/api/feeds", timeout=TIMEOUT)
        assert feeds_resp.status_code == 200
        feeds = feeds_resp.json().get("data", [])
        if not feeds:
            pytest.skip("没有 feed，无法触发 auto_refresh_complete")

        needs_refresh = [
            f for f in feeds
            if f.get("refresh_status") != "refreshing"
            and f.get("refresh_interval", 0) > 0
        ]
        if not needs_refresh:
            pytest.skip("没有需要刷新的 feed，无法触发 auto_refresh_complete")
        return needs_refresh

    def test_auto_refresh_complete_event(self):
        feeds = self._ensure_feed_needs_refresh()

        msg_data = []
        ready = threading.Event()

        def on_message(ws, msg):
            try:
                data = json.loads(msg)
            except (json.JSONDecodeError, TypeError):
                return
            if data.get("type") == "auto_refresh_complete":
                msg_data.append(data)
                ready.set()

        ws = websocket.WebSocketApp(WS_URL, on_message=on_message)
        ws_thread = threading.Thread(target=ws.run_forever, daemon=True)
        ws_thread.start()
        time.sleep(1)

        try:
            resp = _trigger("auto_refresh")
            assert resp.status_code == 200, f"触发 auto_refresh 失败: {resp.status_code}"

            trigger_result = resp.json()
            triggered = trigger_result.get("data", {}).get("summary", {}).get("triggered_feeds", 0)
            if triggered == 0:
                pytest.skip(
                    f"本次没有 feed 被触发刷新（共 {len(feeds)} 个 feed），"
                    "auto_refresh_complete 不会广播。"
                    "可尝试降低 feed refresh_interval 或清除 last_refresh_at 触发。"
                )

            got_event = ready.wait(timeout=90)
            assert got_event, "90s 内未收到 auto_refresh_complete 事件"

            msg = msg_data[0]

            for field in self.REQUIRED_FIELDS:
                assert field in msg, f"消息缺少字段: {field}，实际: {list(msg.keys())}"

            assert isinstance(msg["triggered_feeds"], int)
            assert isinstance(msg["stale_reset_feeds"], int)
            assert isinstance(msg["duration_seconds"], (int, float))
            assert msg["duration_seconds"] >= 0, "duration_seconds 应为非负数"

            ts = msg["timestamp"]
            datetime.fromisoformat(ts)
        finally:
            ws.close()
