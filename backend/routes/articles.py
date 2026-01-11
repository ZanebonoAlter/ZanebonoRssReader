from flask import Blueprint, request, jsonify
from models import Article, Feed, db
from datetime import datetime

articles_bp = Blueprint('articles', __name__, url_prefix='/api/articles')

@articles_bp.route('/stats', methods=['GET'])
def get_articles_stats():
    """Get articles statistics"""
    try:
        total_articles = Article.query.count()
        unread_articles = Article.query.filter_by(read=False).count()
        favorite_articles = Article.query.filter_by(favorite=True).count()

        return jsonify({
            'success': True,
            'data': {
                'total': total_articles,
                'unread': unread_articles,
                'favorite': favorite_articles
            }
        }), 200

    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@articles_bp.route('', methods=['GET'])
def get_articles():
    """Get articles with filters"""
    try:
        page = request.args.get('page', 1, type=int)
        per_page = request.args.get('per_page', 20, type=int)
        feed_id = request.args.get('feed_id', type=int)
        category_id = request.args.get('category_id', type=int)
        uncategorized = request.args.get('uncategorized', type=str)
        read = request.args.get('read', type=str)
        favorite = request.args.get('favorite', type=str)
        search = request.args.get('search', type=str)

        query = Article.query

        # Filters
        if feed_id:
            query = query.filter_by(feed_id=feed_id)

        if category_id:
            query = query.join(Feed).filter(Feed.category_id == category_id)

        if uncategorized is not None and uncategorized.lower() == 'true':
            # Filter articles from feeds without category
            query = query.join(Feed).filter(Feed.category_id == None)

        if read is not None:
            is_read = read.lower() == 'true'
            query = query.filter_by(read=is_read)

        if favorite is not None:
            is_favorite = favorite.lower() == 'true'
            query = query.filter_by(favorite=is_favorite)

        if search:
            search_term = f'%{search}%'
            query = query.filter(
                (Article.title.ilike(search_term)) |
                (Article.description.ilike(search_term))
            )

        # Order by publication date (newest first)
        query = query.order_by(Article.pub_date.desc())

        # If per_page is very large (10000+), return all results without pagination
        if per_page >= 10000:
            articles = query.all()
            return jsonify({
                'success': True,
                'data': [article.to_dict() for article in articles],
                'pagination': {
                    'page': 1,
                    'per_page': len(articles),
                    'total': len(articles),
                    'pages': 1
                }
            }), 200

        # Pagination
        pagination = query.paginate(
            page=page, per_page=per_page, error_out=False
        )

        return jsonify({
            'success': True,
            'data': [article.to_dict() for article in pagination.items],
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

@articles_bp.route('/<int:article_id>', methods=['GET'])
def get_article(article_id):
    """Get a single article"""
    try:
        article = Article.query.get(article_id)
        if not article:
            return jsonify({
                'success': False,
                'error': 'Article not found'
            }), 404

        return jsonify({
            'success': True,
            'data': article.to_dict()
        }), 200

    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@articles_bp.route('/<int:article_id>', methods=['PUT'])
def update_article(article_id):
    """Update article (read/favorite status)"""
    try:
        article = Article.query.get(article_id)
        if not article:
            return jsonify({
                'success': False,
                'error': 'Article not found'
            }), 404

        data = request.get_json()

        if 'read' in data:
            article.read = data['read']

        if 'favorite' in data:
            article.favorite = data['favorite']

        db.session.commit()

        return jsonify({
            'success': True,
            'data': article.to_dict()
        }), 200

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@articles_bp.route('/bulk-update', methods=['PUT'])
def bulk_update_articles():
    """Bulk update articles"""
    try:
        data = request.get_json()

        if not data or not data.get('ids'):
            return jsonify({
                'success': False,
                'error': 'ids is required'
            }), 400

        article_ids = data['ids']
        read = data.get('read')
        favorite = data.get('favorite')

        articles = Article.query.filter(Article.id.in_(article_ids)).all()

        for article in articles:
            if read is not None:
                article.read = read
            if favorite is not None:
                article.favorite = favorite

        db.session.commit()

        return jsonify({
            'success': True,
            'message': f'{len(articles)} articles updated'
        }), 200

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500
