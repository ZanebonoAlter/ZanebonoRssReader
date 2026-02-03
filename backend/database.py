from flask_sqlalchemy import SQLAlchemy
from config import Config

db = SQLAlchemy()

def init_db(app):
    """Initialize database with Flask app"""
    db.init_app(app)
    
    # 应用连接池配置（如果配置中有 SQLALCHEMY_ENGINE_OPTIONS）
    if hasattr(app.config, 'SQLALCHEMY_ENGINE_OPTIONS') or 'SQLALCHEMY_ENGINE_OPTIONS' in app.config:
        # Flask-SQLAlchemy 会自动从配置中读取 SQLALCHEMY_ENGINE_OPTIONS
        # 但我们需要确保配置被正确应用
        engine_options = app.config.get('SQLALCHEMY_ENGINE_OPTIONS', {})
        if engine_options:
            app.logger.info(f'Database connection pool configured: {engine_options}')

    with app.app_context():
        db.create_all()
        app.logger.info('Database initialized successfully')

    return db
