from flask import Blueprint, request, jsonify
from models import Category, db
from datetime import datetime

categories_bp = Blueprint('categories', __name__, url_prefix='/api/categories')

@categories_bp.route('', methods=['GET'])
def get_categories():
    """Get all categories"""
    try:
        categories = Category.query.order_by(Category.name).all()
        return jsonify({
            'success': True,
            'data': [cat.to_dict() for cat in categories]
        }), 200
    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@categories_bp.route('', methods=['POST'])
def create_category():
    """Create a new category"""
    try:
        data = request.get_json()

        if not data or not data.get('name'):
            return jsonify({
                'success': False,
                'error': 'Category name is required'
            }), 400

        # Check if category already exists
        existing = Category.query.filter_by(name=data['name']).first()
        if existing:
            return jsonify({
                'success': False,
                'error': 'Category with this name already exists'
            }), 409

        category = Category(
            name=data['name'],
            slug=data.get('slug', data['name'].lower().replace(' ', '-')),
            icon=data.get('icon', 'folder'),
            color=data.get('color', '#6366f1'),
            description=data.get('description')
        )

        db.session.add(category)
        db.session.commit()

        return jsonify({
            'success': True,
            'data': category.to_dict()
        }), 201

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@categories_bp.route('/<int:category_id>', methods=['PUT'])
def update_category(category_id):
    """Update a category"""
    try:
        category = Category.query.get(category_id)
        if not category:
            return jsonify({
                'success': False,
                'error': 'Category not found'
            }), 404

        data = request.get_json()

        if 'name' in data:
            category.name = data['name']
        if 'slug' in data:
            category.slug = data['slug']
        if 'icon' in data:
            category.icon = data['icon']
        if 'color' in data:
            category.color = data['color']
        if 'description' in data:
            category.description = data['description']

        db.session.commit()

        return jsonify({
            'success': True,
            'data': category.to_dict()
        }), 200

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@categories_bp.route('/<int:category_id>', methods=['DELETE'])
def delete_category(category_id):
    """Delete a category"""
    try:
        category = Category.query.get(category_id)
        if not category:
            return jsonify({
                'success': False,
                'error': 'Category not found'
            }), 404

        db.session.delete(category)
        db.session.commit()

        return jsonify({
            'success': True,
            'message': 'Category deleted successfully'
        }), 200

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500
