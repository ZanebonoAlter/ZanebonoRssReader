from datetime import datetime, timedelta, timezone
from database import db
import hashlib

# Shanghai timezone (UTC+8)
SHANGHAI_TZ = timezone(timedelta(hours=8))

def generate_slug(name):
    """Generate URL slug from name"""
    return hashlib.md5(name.encode()).hexdigest()[:8]

def format_datetime_cst(dt):
    """Format datetime to Shanghai timezone ISO string"""
    if dt is None:
        return None
    # If datetime is naive (no timezone), assume it's UTC
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    # Convert to Shanghai timezone
    dt_cst = dt.astimezone(SHANGHAI_TZ)
    return dt_cst.isoformat()

class Category(db.Model):
    """RSS Feed Category model"""
    __tablename__ = 'categories'

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False, unique=True)
    slug = db.Column(db.String(50), nullable=False, unique=True)
    icon = db.Column(db.String(50), default='folder')
    color = db.Column(db.String(20), default='#6366f1')
    description = db.Column(db.Text)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

    # Relationship with feeds
    feeds = db.relationship('Feed', backref='category', lazy=True, cascade='all, delete-orphan')

    def to_dict(self):
        """Convert category to dictionary"""
        return {
            'id': self.id,
            'name': self.name,
            'slug': self.slug,
            'icon': self.icon,
            'color': self.color,
            'description': self.description,
            'created_at': format_datetime_cst(self.created_at),
            'feed_count': len(self.feeds)
        }

    def __repr__(self):
        return f'<Category {self.name}>'

class Feed(db.Model):
    """RSS Feed model"""
    __tablename__ = 'feeds'

    id = db.Column(db.Integer, primary_key=True)
    title = db.Column(db.String(200), nullable=False)
    description = db.Column(db.Text)
    url = db.Column(db.String(500), nullable=False, unique=True)
    category_id = db.Column(db.Integer, db.ForeignKey('categories.id'), nullable=True)
    icon = db.Column(db.String(50), default='rss')
    color = db.Column(db.String(20), default='#8b5cf6')
    last_updated = db.Column(db.DateTime)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    max_articles = db.Column(db.Integer, default=100)  # Maximum articles to keep
    refresh_interval = db.Column(db.Integer, default=60)  # Auto-refresh interval in minutes
    refresh_status = db.Column(db.String(20), default='idle')  # idle, refreshing, success, error
    refresh_error = db.Column(db.Text)  # Last refresh error message
    last_refresh_at = db.Column(db.DateTime)  # Last refresh attempt time
    ai_summary_enabled = db.Column(db.Boolean, default=True)  # Include in AI summaries

    # Relationship with articles
    articles = db.relationship('Article', backref='feed', lazy=True, cascade='all, delete-orphan')

    def to_dict(self, include_stats=False):
        """Convert feed to dictionary"""
        data = {
            'id': self.id,
            'title': self.title,
            'description': self.description,
            'url': self.url,
            'category_id': self.category_id,
            'icon': self.icon,
            'color': self.color,
            'last_updated': format_datetime_cst(self.last_updated),
            'created_at': format_datetime_cst(self.created_at),
            'max_articles': self.max_articles,
            'refresh_interval': self.refresh_interval,
            'refresh_status': self.refresh_status,
            'refresh_error': self.refresh_error,
            'last_refresh_at': format_datetime_cst(self.last_refresh_at),
            'ai_summary_enabled': self.ai_summary_enabled
        }

        if include_stats:
            data['article_count'] = len(self.articles)
            data['unread_count'] = sum(1 for a in self.articles if not a.read)

        return data

    def __repr__(self):
        return f'<Feed {self.title}>'

class Article(db.Model):
    """RSS Article model"""
    __tablename__ = 'articles'

    id = db.Column(db.Integer, primary_key=True)
    feed_id = db.Column(db.Integer, db.ForeignKey('feeds.id'), nullable=False)
    title = db.Column(db.String(500), nullable=False)
    description = db.Column(db.Text)
    content = db.Column(db.Text)
    link = db.Column(db.String(1000))
    pub_date = db.Column(db.DateTime)
    author = db.Column(db.String(200))
    read = db.Column(db.Boolean, default=False)
    favorite = db.Column(db.Boolean, default=False)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

    def to_dict(self):
        """Convert article to dictionary"""
        return {
            'id': self.id,
            'feed_id': self.feed_id,
            'title': self.title,
            'description': self.description,
            'content': self.content,
            'link': self.link,
            'pub_date': format_datetime_cst(self.pub_date),
            'author': self.author,
            'read': self.read,
            'favorite': self.favorite,
            'created_at': format_datetime_cst(self.created_at)
        }

    def __repr__(self):
        return f'<Article {self.title}>'

class AISummary(db.Model):
    """AI Summary model for category/article aggregation summaries"""
    __tablename__ = 'ai_summaries'

    id = db.Column(db.Integer, primary_key=True)
    category_id = db.Column(db.Integer, db.ForeignKey('categories.id'), nullable=True)
    title = db.Column(db.String(200), nullable=False)  # Summary title
    summary = db.Column(db.Text, nullable=False)  # Main summary content
    key_points = db.Column(db.Text)  # JSON array of key points
    articles = db.Column(db.Text)  # JSON array of article IDs included
    article_count = db.Column(db.Integer, default=0)  # Number of articles summarized
    time_range = db.Column(db.Integer, default=180)  # Time range in minutes (default 3 hours)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationship with category
    category = db.relationship('Category', backref='ai_summaries')

    def to_dict(self):
        """Convert AI summary to dictionary"""
        return {
            'id': self.id,
            'category_id': self.category_id,
            'title': self.title,
            'summary': self.summary,
            'key_points': self.key_points,
            'articles': self.articles,
            'article_count': self.article_count,
            'time_range': self.time_range,
            'created_at': format_datetime_cst(self.created_at),
            'updated_at': format_datetime_cst(self.updated_at),
            'category_name': self.category.name if self.category else '全部分类'
        }

    def __repr__(self):
        return f'<AISummary {self.title}>'

class SchedulerTask(db.Model):
    """Scheduler task execution tracking model"""
    __tablename__ = 'scheduler_tasks'

    # Primary identification
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(50), nullable=False, unique=True, index=True)  # 'auto_refresh', 'auto_summary'
    description = db.Column(db.String(200))

    # Execution timing
    check_interval = db.Column(db.Integer, nullable=False, default=60)  # Check interval in seconds
    last_execution_time = db.Column(db.DateTime)  # Last time the task executed
    next_execution_time = db.Column(db.DateTime)  # Calculated next execution time

    # Current status
    status = db.Column(db.String(20), default='idle', index=True)  # idle, running, success, error
    last_error = db.Column(db.Text)  # Last error message
    last_error_time = db.Column(db.DateTime)  # When the last error occurred

    # Statistics
    total_executions = db.Column(db.Integer, default=0)  # Total number of executions
    successful_executions = db.Column(db.Integer, default=0)  # Successful executions
    failed_executions = db.Column(db.Integer, default=0)  # Failed executions
    consecutive_failures = db.Column(db.Integer, default=0)  # Consecutive failure count

    # Additional metadata
    last_execution_duration = db.Column(db.Float)  # Last execution duration in seconds
    last_execution_result = db.Column(db.Text)  # Additional result info (JSON or text)

    # Timestamps
    created_at = db.Column(db.DateTime, default=datetime.utcnow, nullable=False)
    updated_at = db.Column(db.DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)

    def to_dict(self):
        """Convert scheduler task to dictionary"""
        success_rate = 0
        if self.total_executions > 0:
            success_rate = (self.successful_executions / self.total_executions) * 100

        return {
            'id': self.id,
            'name': self.name,
            'description': self.description,
            'check_interval': self.check_interval,
            'last_execution_time': format_datetime_cst(self.last_execution_time),
            'next_execution_time': format_datetime_cst(self.next_execution_time),
            'status': self.status,
            'last_error': self.last_error,
            'last_error_time': format_datetime_cst(self.last_error_time),
            'total_executions': self.total_executions,
            'successful_executions': self.successful_executions,
            'failed_executions': self.failed_executions,
            'consecutive_failures': self.consecutive_failures,
            'last_execution_duration': self.last_execution_duration,
            'last_execution_result': self.last_execution_result,
            'created_at': format_datetime_cst(self.created_at),
            'updated_at': format_datetime_cst(self.updated_at),
            'success_rate': success_rate
        }

    def __repr__(self):
        return f'<SchedulerTask {self.name}>'

class AISettings(db.Model):
    """AI Configuration Settings model"""
    __tablename__ = 'ai_settings'

    id = db.Column(db.Integer, primary_key=True)
    key = db.Column(db.String(100), nullable=False, unique=True, index=True)  # e.g., 'summary_config', 'podcast_config'
    value = db.Column(db.Text, nullable=True)  # JSON string containing the configuration
    description = db.Column(db.String(200))  # Human-readable description
    created_at = db.Column(db.DateTime, default=datetime.utcnow, nullable=False)
    updated_at = db.Column(db.DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)

    def to_dict(self):
        """Convert AI settings to dictionary"""
        import json
        return {
            'id': self.id,
            'key': self.key,
            'value': json.loads(self.value) if self.value else None,
            'description': self.description,
            'created_at': format_datetime_cst(self.created_at),
            'updated_at': format_datetime_cst(self.updated_at)
        }

    def __repr__(self):
        return f'<AISettings {self.key}>'
