"""
Scheduler status and control routes
"""
from flask import Blueprint, request, jsonify
from models import SchedulerTask, db
from datetime import datetime, timedelta

schedulers_bp = Blueprint('schedulers', __name__, url_prefix='/api/schedulers')


@schedulers_bp.route('/status', methods=['GET'])
def get_all_schedulers_status():
    """Get status of all schedulers"""
    try:
        schedulers = SchedulerTask.query.all()
        return jsonify({
            'success': True,
            'data': [scheduler.to_dict() for scheduler in schedulers]
        }), 200
    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@schedulers_bp.route('/<string:name>/status', methods=['GET'])
def get_scheduler_status(name):
    """Get status of a specific scheduler"""
    try:
        scheduler = SchedulerTask.query.filter_by(name=name).first()
        if not scheduler:
            return jsonify({
                'success': False,
                'error': f'Scheduler "{name}" not found'
            }), 404

        return jsonify({
            'success': True,
            'data': scheduler.to_dict()
        }), 200
    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@schedulers_bp.route('/<string:name>/trigger', methods=['POST'])
def trigger_scheduler(name):
    """
    Manually trigger a scheduler execution
    This updates the next_execution_time to now, causing immediate execution
    """
    try:
        scheduler = SchedulerTask.query.filter_by(name=name).first()
        if not scheduler:
            return jsonify({
                'success': False,
                'error': f'Scheduler "{name}" not found'
            }), 404

        # Set next execution to now
        scheduler.next_execution_time = datetime.utcnow()
        scheduler.updated_at = datetime.utcnow()
        db.session.commit()

        return jsonify({
            'success': True,
            'message': f'Scheduler "{name}" triggered for immediate execution',
            'data': scheduler.to_dict()
        }), 200
    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@schedulers_bp.route('/<string:name>/reset', methods=['POST'])
def reset_scheduler_stats(name):
    """Reset statistics for a specific scheduler"""
    try:
        scheduler = SchedulerTask.query.filter_by(name=name).first()
        if not scheduler:
            return jsonify({
                'success': False,
                'error': f'Scheduler "{name}" not found'
            }), 404

        # Reset statistics
        scheduler.total_executions = 0
        scheduler.successful_executions = 0
        scheduler.failed_executions = 0
        scheduler.consecutive_failures = 0
        scheduler.last_error = None
        scheduler.last_error_time = None
        scheduler.updated_at = datetime.utcnow()
        db.session.commit()

        return jsonify({
            'success': True,
            'message': f'Statistics reset for scheduler "{name}"',
            'data': scheduler.to_dict()
        }), 200
    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@schedulers_bp.route('/<string:name>/interval', methods=['PUT'])
def update_scheduler_interval(name):
    """Update check interval for a specific scheduler"""
    try:
        data = request.get_json()
        interval = data.get('interval')

        if not interval or not isinstance(interval, int) or interval <= 0:
            return jsonify({
                'success': False,
                'error': 'Valid interval (positive integer) is required'
            }), 400

        scheduler = SchedulerTask.query.filter_by(name=name).first()
        if not scheduler:
            return jsonify({
                'success': False,
                'error': f'Scheduler "{name}" not found'
            }), 404

        # Update interval
        old_interval = scheduler.check_interval
        scheduler.check_interval = interval
        scheduler.updated_at = datetime.utcnow()
        db.session.commit()

        return jsonify({
            'success': True,
            'message': f'Interval updated from {old_interval}s to {interval}s',
            'data': scheduler.to_dict()
        }), 200
    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500
