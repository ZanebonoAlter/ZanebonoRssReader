"""Base class for scheduler state management"""
import logging
from datetime import datetime, timedelta
from models import SchedulerTask, db
from models import format_datetime_cst

logger = logging.getLogger(__name__)


class SchedulerStateManager:
    """
    Base class for managing scheduler state persistence

    Provides methods to:
    - Initialize scheduler state in database
    - Update execution status before/after runs
    - Track execution statistics
    - Handle error recording
    """

    def __init__(self, scheduler_name, description, default_interval=60):
        """
        Initialize the state manager

        Args:
            scheduler_name: Unique identifier (e.g., 'auto_refresh', 'auto_summary')
            description: Human-readable description
            default_interval: Default check interval in seconds
        """
        self.scheduler_name = scheduler_name
        self.description = description
        self.default_interval = default_interval

    def initialize_state(self, app, check_interval=None):
        """
        Initialize or load scheduler state from database
        Creates record if doesn't exist

        Returns: SchedulerTask object
        """
        try:
            with app.app_context():
                # Try to get existing task
                task = SchedulerTask.query.filter_by(name=self.scheduler_name).first()

                if not task:
                    # Create new task record
                    interval = check_interval or self.default_interval
                    task = SchedulerTask(
                        name=self.scheduler_name,
                        description=self.description,
                        check_interval=interval,
                        status='idle',
                        next_execution_time=datetime.utcnow() + timedelta(seconds=interval)
                    )
                    db.session.add(task)
                    db.session.commit()
                    logger.info(f"Created new scheduler state for '{self.scheduler_name}'")
                else:
                    # Update interval if provided
                    if check_interval and task.check_interval != check_interval:
                        task.check_interval = check_interval
                        db.session.commit()
                        logger.info(f"Updated check interval for '{self.scheduler_name}' to {check_interval}s")

                return task

        except Exception as e:
            logger.error(f"Failed to initialize scheduler state: {e}", exc_info=True)
            return None

    def should_execute_immediately(self, task):
        """
        Check if task should execute immediately on startup
        (i.e., it's overdue)

        Args:
            task: SchedulerTask object

        Returns: bool
        """
        if not task or not task.next_execution_time:
            return False

        now = datetime.utcnow()
        if now >= task.next_execution_time:
            logger.info(f"Task '{self.scheduler_name}' is overdue, should execute immediately")
            return True
        return False

    def pre_execute(self, task):
        """
        Update state before execution
        Sets status to 'running'
        NOTE: Must be called within an active app_context

        Args:
            task: SchedulerTask object

        Returns: bool (success)
        """
        try:
            task.status = 'running'
            task.updated_at = datetime.utcnow()
            db.session.commit()
            logger.debug(f"Task '{self.scheduler_name}' status set to 'running'")
            return True
        except Exception as e:
            logger.error(f"Failed to update pre-execution state: {e}", exc_info=True)
            return False

    def post_execute_success(self, task, duration=None, result=None):
        """
        Update state after successful execution
        NOTE: Must be called within an active app_context

        Args:
            task: SchedulerTask object
            duration: Execution duration in seconds
            result: Additional result info

        Returns: bool (success)
        """
        try:
            now = datetime.utcnow()

            # Update status
            task.status = 'success'
            task.last_execution_time = now
            task.next_execution_time = now + timedelta(seconds=task.check_interval)
            task.last_error = None
            task.last_error_time = None
            task.consecutive_failures = 0

            # Update statistics
            task.total_executions += 1
            task.successful_executions += 1

            # Update optional fields
            if duration is not None:
                task.last_execution_duration = duration
            if result is not None:
                task.last_execution_result = str(result)[:1000]  # Limit to 1000 chars

            task.updated_at = now
            db.session.commit()

            duration_str = f"{duration:.2f}s" if duration is not None else 'N/A'
            logger.info(
                f"Task '{self.scheduler_name}' completed successfully "
                f"(duration: {duration_str}, "
                f"total: {task.total_executions}, "
                f"success: {task.successful_executions})"
            )
            return True

        except Exception as e:
            logger.error(f"Failed to update post-execution success state: {e}", exc_info=True)
            return False

    def post_execute_error(self, task, error_message):
        """
        Update state after failed execution
        NOTE: Must be called within an active app_context

        Args:
            task: SchedulerTask object
            error_message: Error message

        Returns: bool (success)
        """
        try:
            now = datetime.utcnow()

            # Update status
            task.status = 'error'
            task.last_execution_time = now
            task.next_execution_time = now + timedelta(seconds=task.check_interval)
            task.last_error = str(error_message)[:2000]  # Limit to 2000 chars
            task.last_error_time = now

            # Update statistics
            task.total_executions += 1
            task.failed_executions += 1
            task.consecutive_failures += 1

            task.updated_at = now
            db.session.commit()

            logger.warning(
                f"Task '{self.scheduler_name}' failed: {error_message} "
                f"(consecutive failures: {task.consecutive_failures})"
            )
            return True

        except Exception as e:
            logger.error(f"Failed to update post-execution error state: {e}", exc_info=True)
            return False

    def get_status(self, app):
        """
        Get current scheduler status from database

        Args:
            app: Flask app

        Returns: dict
        """
        try:
            with app.app_context():
                task = SchedulerTask.query.filter_by(name=self.scheduler_name).first()
                if task:
                    return task.to_dict()
                else:
                    return {
                        'name': self.scheduler_name,
                        'status': 'not_initialized',
                        'error': 'Scheduler state not found in database'
                    }
        except Exception as e:
            logger.error(f"Failed to get scheduler status: {e}", exc_info=True)
            return {
                'name': self.scheduler_name,
                'status': 'error',
                'error': str(e)
            }
