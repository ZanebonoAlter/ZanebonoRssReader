from flask import Blueprint, request, jsonify
from models import Feed, Category, Article, db
from datetime import datetime
import feedparser
import hashlib
import threading

feeds_bp = Blueprint('feeds', __name__, url_prefix='/api/feeds')

def parse_feed_url(url):
    """Parse RSS feed from URL"""
    try:
        feed_data = feedparser.parse(url)

        if feed_data.bozo and feed_data.bozo_exception:
            raise Exception(f"Invalid feed URL: {str(feed_data.bozo_exception)}")

        if not feed_data.feed:
            raise Exception("Unable to parse feed")

        return feed_data
    except Exception as e:
        raise Exception(f"Error parsing feed: {str(e)}")

@feeds_bp.route('', methods=['GET'])
def get_feeds():
    """Get all feeds"""
    try:
        page = request.args.get('page', 1, type=int)
        per_page = request.args.get('per_page', 20, type=int)
        category_id = request.args.get('category_id', type=int)
        uncategorized = request.args.get('uncategorized', type=str)

        query = Feed.query

        if category_id:
            query = query.filter_by(category_id=category_id)

        if uncategorized is not None and uncategorized.lower() == 'true':
            # Filter feeds without category
            query = query.filter_by(category_id=None)

        # If per_page is very large (10000+), return all results without pagination
        if per_page >= 10000:
            feeds = query.order_by(Feed.title).all()
            return jsonify({
                'success': True,
                'data': [feed.to_dict(include_stats=True) for feed in feeds],
                'pagination': {
                    'page': 1,
                    'per_page': len(feeds),
                    'total': len(feeds),
                    'pages': 1
                }
            }), 200

        # Pagination
        pagination = query.order_by(Feed.title).paginate(
            page=page, per_page=per_page, error_out=False
        )

        return jsonify({
            'success': True,
            'data': [feed.to_dict(include_stats=True) for feed in pagination.items],
            'pagination': {
                'page': page,
                'per_page': per_page,
                'total': pagination.total,
                'pages': pagination.pages
            }
        }), 200

    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@feeds_bp.route('', methods=['POST'])
def create_feed():
    """Create a new feed"""
    try:
        data = request.get_json()

        if not data or not data.get('url'):
            return jsonify({
                'success': False,
                'error': 'Feed URL is required'
            }), 400

        # Check if feed already exists
        existing = Feed.query.filter_by(url=data['url']).first()
        if existing:
            return jsonify({
                'success': False,
                'error': 'Feed with this URL already exists'
            }), 409

        # Parse feed to get metadata
        feed_data = parse_feed_url(data['url'])

        # Extract icon from feed
        feed_icon = data.get('icon')
        if not feed_icon:
            # Try to get image from feed
            if feed_data.feed.get('image'):
                feed_icon = feed_data.feed.get('image', {}).get('href')

            # Fallback to favicon from first article link
            if not feed_icon and feed_data.entries and len(feed_data.entries) > 0:
                first_entry_link = feed_data.entries[0].get('link')
                if first_entry_link:
                    try:
                        from urllib.parse import urlparse
                        parsed = urlparse(first_entry_link)
                        feed_icon = f"{parsed.scheme}://{parsed.netloc}/favicon.ico"
                    except Exception:
                        pass

            # Final fallback to feed URL's favicon
            if not feed_icon:
                try:
                    from urllib.parse import urlparse
                    parsed_url = urlparse(data['url'])
                    feed_icon = f"{parsed_url.scheme}://{parsed_url.netloc}/favicon.ico"
                except Exception:
                    feed_icon = 'rss'

        # Create feed
        feed = Feed(
            title=data.get('title') or feed_data.feed.get('title', 'Untitled Feed'),
            description=data.get('description') or feed_data.feed.get('description'),
            url=data['url'],
            category_id=data.get('category_id'),
            icon=feed_icon or 'rss',
            color=data.get('color', '#8b5cf6'),
            last_updated=datetime.utcnow(),
            max_articles=data.get('max_articles', 100),
            refresh_interval=data.get('refresh_interval', 60)
        )

        db.session.add(feed)
        db.session.flush()

        # Import articles from feed
        articles_added = 0
        for entry in feed_data.entries[:feed.max_articles]:  # Limit to max_articles
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

        db.session.commit()

        return jsonify({
            'success': True,
            'data': feed.to_dict(include_stats=True),
            'message': f'Feed created with {articles_added} articles'
        }), 201

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@feeds_bp.route('/<int:feed_id>', methods=['PUT'])
def update_feed(feed_id):
    """Update a feed"""
    try:
        feed = Feed.query.get(feed_id)
        if not feed:
            return jsonify({
                'success': False,
                'error': 'Feed not found'
            }), 404

        data = request.get_json()

        if 'title' in data:
            feed.title = data['title']
        if 'description' in data:
            feed.description = data['description']
        if 'url' in data:
            feed.url = data['url']
        if 'category_id' in data:
            feed.category_id = data['category_id']
        if 'icon' in data:
            feed.icon = data['icon']
        if 'color' in data:
            feed.color = data['color']
        if 'max_articles' in data:
            feed.max_articles = data['max_articles']
        if 'refresh_interval' in data:
            feed.refresh_interval = data['refresh_interval']
        if 'ai_summary_enabled' in data:
            feed.ai_summary_enabled = data['ai_summary_enabled']

        db.session.commit()

        return jsonify({
            'success': True,
            'data': feed.to_dict(include_stats=True)
        }), 200

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@feeds_bp.route('/<int:feed_id>', methods=['DELETE'])
def delete_feed(feed_id):
    """Delete a feed"""
    try:
        feed = Feed.query.get(feed_id)
        if not feed:
            return jsonify({
                'success': False,
                'error': 'Feed not found'
            }), 404

        db.session.delete(feed)
        db.session.commit()

        return jsonify({
            'success': True,
            'message': 'Feed deleted successfully'
        }), 200

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@feeds_bp.route('/<int:feed_id>/refresh', methods=['POST'])
def refresh_feed(feed_id):
    """Refresh a feed asynchronously"""
    try:
        from flask import current_app

        feed = Feed.query.get(feed_id)
        if not feed:
            return jsonify({
                'success': False,
                'error': 'Feed not found'
            }), 404

        # Update status to refreshing
        feed.refresh_status = 'refreshing'
        feed.last_refresh_at = datetime.utcnow()
        feed.refresh_error = None
        db.session.commit()

        # Start refresh in background thread
        app = current_app._get_current_object()
        thread = threading.Thread(target=refresh_feed_worker, args=(feed_id, app))
        thread.daemon = True
        thread.start()

        return jsonify({
            'success': True,
            'message': f'Started refreshing feed "{feed.title}" in background'
        }), 202  # 202 Accepted

    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@feeds_bp.route('/fetch', methods=['POST'])
def fetch_feed():
    """Fetch RSS feed metadata from URL"""
    try:
        data = request.get_json()

        if not data or not data.get('url'):
            return jsonify({
                'success': False,
                'error': 'Feed URL is required'
            }), 400

        feed_data = parse_feed_url(data['url'])

        return jsonify({
            'success': True,
            'data': {
                'title': feed_data.feed.get('title'),
                'description': feed_data.feed.get('description'),
                'link': feed_data.feed.get('link')
            }
        }), 200

    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

def refresh_feed_worker(feed_id, app):
    """Worker function to refresh a single feed"""
    with app.app_context():
        try:
            feed = Feed.query.get(feed_id)
            if not feed:
                return

            # Update status to refreshing
            feed.refresh_status = 'refreshing'
            feed.last_refresh_at = datetime.utcnow()
            feed.refresh_error = None
            db.session.commit()

            # Parse feed to get latest articles
            feed_data = parse_feed_url(feed.url)

            # Update feed metadata
            feed.title = feed_data.feed.get('title', feed.title)
            feed.description = feed_data.feed.get('description', feed.description)
            feed.last_updated = datetime.utcnow()

            # Update icon if available and current icon is 'rss' or empty
            from urllib.parse import urlparse

            # Try to get icon from feed image
            new_icon = None
            if feed_data.feed.get('image'):
                new_icon = feed_data.feed.get('image', {}).get('href')

            # If no icon in feed, or current icon is 'rss' or empty, try favicon
            if not new_icon or feed.icon in ['rss', '', None]:
                # Try to get favicon from the first article's link
                if feed_data.entries and len(feed_data.entries) > 0:
                    first_entry_link = feed_data.entries[0].get('link')
                    if first_entry_link:
                        try:
                            parsed = urlparse(first_entry_link)
                            new_icon = f"{parsed.scheme}://{parsed.netloc}/favicon.ico"
                        except Exception as e:
                            print(f"Error parsing URL for favicon: {e}")

                # Fallback to feed URL
                if not new_icon:
                    try:
                        parsed = urlparse(feed.url)
                        new_icon = f"{parsed.scheme}://{parsed.netloc}/favicon.ico"
                    except Exception as e:
                        print(f"Error parsing feed URL for favicon: {e}")

            # Update icon if we found a new one and current is 'rss' or empty
            if new_icon and feed.icon in ['rss', '', None]:
                feed.icon = new_icon

            # Get existing article links for this feed (for faster lookup)
            existing_links = set(article.link for article in Article.query.filter_by(feed_id=feed.id).all())

            # Import new articles
            articles_added = 0
            entries_seen = 0
            for entry in feed_data.entries:
                entries_seen += 1
                try:
                    link = entry.get('link')

                    # Skip if no link
                    if not link:
                        continue

                    # Skip if article already exists
                    if link in existing_links:
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

                    # Limit to max_articles
                    if articles_added >= feed.max_articles:
                        break

                except Exception as entry_error:
                    print(f"Error processing entry in feed {feed.id}: {str(entry_error)}")
                    continue

            # Clean up old articles
            all_articles = Article.query.filter_by(feed_id=feed.id).order_by(Article.pub_date.desc()).all()
            if len(all_articles) > feed.max_articles:
                articles_to_delete = all_articles[feed.max_articles:]
                for article in articles_to_delete:
                    db.session.delete(article)

            # Update status to success
            feed.refresh_status = 'success'
            db.session.commit()

            print(f"Feed {feed.id} ({feed.title}) refreshed: saw {entries_seen} entries, added {articles_added} new articles (had {len(existing_links)} existing)")

        except Exception as e:
            # Update status to error
            try:
                feed = Feed.query.get(feed_id)
                if feed:
                    feed.refresh_status = 'error'
                    feed.refresh_error = str(e)
                    db.session.commit()
            except:
                pass
            print(f"Error refreshing feed {feed_id}: {str(e)}")
        finally:
            # 确保数据库会话被正确关闭，释放连接
            # 在多线程环境中，显式关闭会话可以防止连接泄漏
            db.session.remove()


@feeds_bp.route('/refresh-all', methods=['POST'])
def refresh_all_feeds():
    """Refresh all feeds asynchronously"""
    try:
        from flask import current_app

        feeds = Feed.query.all()

        if not feeds:
            return jsonify({
                'success': True,
                'message': 'No feeds to refresh'
            }), 200

        # Mark all feeds as refreshing and start async refresh
        for feed in feeds:
            feed.refresh_status = 'refreshing'
            feed.last_refresh_at = datetime.utcnow()
            feed.refresh_error = None

        db.session.commit()

        # Start refresh in background thread for each feed
        app = current_app._get_current_object()
        for feed in feeds:
            thread = threading.Thread(target=refresh_feed_worker, args=(feed.id, app))
            thread.daemon = True  # Daemon thread will not prevent app from exiting
            thread.start()

        return jsonify({
            'success': True,
            'message': f'Started refreshing {len(feeds)} feeds in background'
        }), 202  # 202 Accepted

    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500
