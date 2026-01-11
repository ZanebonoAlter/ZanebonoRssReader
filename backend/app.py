from flask import Flask, jsonify, request
from flask_cors import CORS
from config import Config
from database import init_db, db
import logging

# Import blueprints
from routes.categories import categories_bp
from routes.feeds import feeds_bp
from routes.articles import articles_bp
from routes.opml import opml_bp
from routes.ai import ai_bp
from routes.summaries import summaries_bp
from routes.schedulers import schedulers_bp

def create_app(config_class=Config):
    """Create and configure Flask application"""
    app = Flask(__name__)
    app.config.from_object(config_class)

    # Setup logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Initialize CORS
    CORS(app, resources={
        r"/api/*": {
            "origins": Config.CORS_ORIGINS,
            "methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
            "allow_headers": ["Content-Type", "Authorization"]
        }
    })

    # Initialize database
    init_db(app)

    # Initialize background tasks (import to start worker thread)
    import tasks
    app.logger.info("Background tasks initialized")

    # Initialize auto-refresh scheduler
    from auto_refresh import init_scheduler
    init_scheduler(app)
    app.logger.info("Auto-refresh scheduler initialized")

    # Initialize auto-summary scheduler
    from auto_summary import init_auto_summary_scheduler
    init_auto_summary_scheduler(app)
    app.logger.info("Auto-summary scheduler initialized")

    # Register blueprints
    app.register_blueprint(categories_bp)
    app.register_blueprint(feeds_bp)
    app.register_blueprint(articles_bp)
    app.register_blueprint(opml_bp)
    app.register_blueprint(ai_bp)
    app.register_blueprint(summaries_bp)
    app.register_blueprint(schedulers_bp)

    # Root endpoint
    @app.route('/')
    def index():
        return jsonify({
            'name': 'RSS Reader API',
            'version': '1.0.0',
            'endpoints': {
                'categories': '/api/categories',
                'feeds': '/api/feeds',
                'articles': '/api/articles',
                'opml': {
                    'import': 'POST /api/import-opml',
                    'export': 'GET /api/export-opml'
                }
            }
        })

    # Health check endpoint
    @app.route('/health')
    def health():
        return jsonify({
            'status': 'healthy',
            'database': 'connected'
        }), 200

    # Task status endpoint
    @app.route('/api/tasks/status')
    def task_status():
        from tasks import get_task_status
        return jsonify({
            'success': True,
            'data': get_task_status()
        }), 200

    # Auto-refresh scheduler status endpoint
    @app.route('/api/auto-refresh/status')
    def auto_refresh_status():
        from auto_refresh import get_scheduler
        from models import Feed
        from datetime import datetime
        
        scheduler = get_scheduler()
        status = scheduler.get_status() if scheduler else {'running': False}
        
        # Add diagnostic information
        with app.app_context():
            feeds_with_interval = Feed.query.filter(Feed.refresh_interval > 0).all()
            feeds_info = []
            now = datetime.utcnow()
            
            for feed in feeds_with_interval:
                time_since_refresh = None
                if feed.last_refresh_at:
                    time_since_refresh = int((now - feed.last_refresh_at).total_seconds() / 60)
                
                feeds_info.append({
                    'id': feed.id,
                    'title': feed.title,
                    'refresh_interval': feed.refresh_interval,
                    'refresh_status': feed.refresh_status,
                    'last_refresh_at': feed.last_refresh_at.isoformat() if feed.last_refresh_at else None,
                    'minutes_since_refresh': time_since_refresh,
                    'needs_refresh': (
                        feed.last_refresh_at is None or
                        (now - feed.last_refresh_at).total_seconds() >= feed.refresh_interval * 60
                    ) if feed.last_refresh_at else True
                })
        
        status['feeds'] = feeds_info
        status['total_feeds_with_interval'] = len(feeds_info)
        
        return jsonify({
            'success': True,
            'data': status
        }), 200

    # Auto-summary scheduler status endpoint
    @app.route('/api/auto-summary/status')
    def auto_summary_status():
        from auto_summary import get_auto_summary_scheduler
        scheduler = get_auto_summary_scheduler()
        return jsonify({
            'success': True,
            'data': scheduler.get_status() if scheduler else {'running': False}
        }), 200

    # Auto-summary scheduler config update endpoint
    @app.route('/api/auto-summary/config', methods=['POST'])
    def auto_summary_config():
        from auto_summary import update_auto_summary_config
        try:
            data = request.get_json()
            base_url = data.get('base_url')
            api_key = data.get('api_key')
            model = data.get('model')

            if not api_key:
                return jsonify({
                    'success': False,
                    'error': 'API key is required'
                }), 400

            update_auto_summary_config(
                base_url or 'https://api.openai.com/v1',
                api_key,
                model or 'gpt-4o-mini'
            )

            return jsonify({
                'success': True,
                'message': 'Auto-summary configuration updated'
            }), 200

        except Exception as e:
            return jsonify({
                'success': False,
                'error': str(e)
            }), 500

    # Error handlers
    @app.errorhandler(404)
    def not_found(error):
        return jsonify({
            'success': False,
            'error': 'Resource not found'
        }), 404

    @app.errorhandler(500)
    def internal_error(error):
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': 'Internal server error'
        }), 500

    @app.errorhandler(400)
    def bad_request(error):
        return jsonify({
            'success': False,
            'error': 'Bad request'
        }), 400

    return app

if __name__ == '__main__':
    app = create_app()
    app.run(host='0.0.0.0', port=5000, debug=True)
