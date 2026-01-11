"""Background tasks for async feed processing"""
import threading
import time
from datetime import datetime
from models import Feed, Article, db
from routes.feeds import parse_feed_url
import queue

# Task queue
task_queue = queue.Queue()
task_lock = threading.Lock()
active_tasks = {}

def add_feed_to_queue(feed_id):
    """Add a feed to the update queue"""
    with task_lock:
        if feed_id not in active_tasks:
            active_tasks[feed_id] = 'queued'
            task_queue.put(feed_id)
            print(f"[Task] Added feed {feed_id} to queue")
            return True
        return False

def update_feed_metadata(feed_id):
    """Update feed metadata and fetch articles - runs in background thread"""
    try:
        print(f"[Task] Starting update for feed {feed_id}")

        # Get feed from database (new session per thread)
        feed = Feed.query.get(feed_id)
        if not feed:
            print(f"[Task] Feed {feed_id} not found")
            return

        # Parse feed to get latest articles
        feed_data = parse_feed_url(feed.url)

        # Update feed metadata
        feed.title = feed_data.feed.get('title', feed.title)
        feed.description = feed_data.feed.get('description', feed.description)
        feed.last_updated = datetime.utcnow()

        # Update icon if available
        if feed_data.feed.get('image'):
            feed.icon = feed_data.feed.get('image', {}).get('href') or feed.icon

        # Import new articles
        articles_added = 0
        for entry in feed_data.entries[:feed.max_articles]:
            # Check if article already exists
            link = entry.get('link')
            if link and Article.query.filter_by(feed_id=feed.id, link=link).first():
                continue

            article = Article(
                feed_id=feed.id,
                title=entry.get('title', 'No title'),
                description=entry.get('description') or entry.get('summary'),
                content=entry.get('content', [{}])[0].get('value') if entry.get('content') else None,
                link=link,
                pub_date=datetime(*entry.published_parsed[:6]) if hasattr(entry, 'published_parsed') and entry.published_parsed else datetime.utcnow(),
                author=entry.get('author')
            )
            db.session.add(article)
            articles_added += 1

        # Clean up old articles if exceeding max_articles
        all_articles = Article.query.filter_by(feed_id=feed.id).order_by(Article.pub_date.desc()).all()
        if len(all_articles) > feed.max_articles:
            articles_to_delete = all_articles[feed.max_articles:]
            for article in articles_to_delete:
                db.session.delete(article)

        db.session.commit()
        print(f"[Task] Completed update for feed {feed_id}: {articles_added} new articles")

    except Exception as e:
        db.session.rollback()
        print(f"[Task] Error updating feed {feed_id}: {str(e)}")
    finally:
        with task_lock:
            if feed_id in active_tasks:
                del active_tasks[feed_id]

def worker():
    """Background worker thread"""
    while True:
        try:
            # Get feed ID from queue
            feed_id = task_queue.get()

            with task_lock:
                active_tasks[feed_id] = 'processing'

            # Update feed
            update_feed_metadata(feed_id)

            # Mark task as done
            task_queue.task_done()

            # Small delay to avoid overwhelming servers
            time.sleep(1)

        except Exception as e:
            print(f"[Task Worker] Error: {str(e)}")
            task_queue.task_done()

# Start worker thread
worker_thread = threading.Thread(target=worker, daemon=True)
worker_thread.start()

print("[Task] Background worker started")


def enqueue_feeds(feed_ids):
    """Enqueue multiple feeds for background processing"""
    enqueued = 0
    for feed_id in feed_ids:
        if add_feed_to_queue(feed_id):
            enqueued += 1
    return enqueued


def get_task_status():
    """Get current task status"""
    with task_lock:
        return {
            'queue_size': task_queue.qsize(),
            'active_tasks': len(active_tasks),
            'tasks': list(active_tasks.keys())
        }
