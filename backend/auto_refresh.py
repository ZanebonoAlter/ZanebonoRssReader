"""Auto-refresh scheduler for RSS feeds"""
import threading
import time
import logging
from datetime import datetime, timedelta
from models import Feed, db, SchedulerTask
from routes.feeds import refresh_feed_worker
from flask import current_app
from scheduler_base import SchedulerStateManager

logger = logging.getLogger(__name__)
# Set logger level to DEBUG to see detailed logs
logger.setLevel(logging.DEBUG)

class AutoRefreshScheduler:
    """Background scheduler to auto-refresh feeds based on their refresh_interval"""

    def __init__(self, check_interval=60):
        """
        Initialize the scheduler

        Args:
            check_interval: How often to check for feeds that need refreshing (seconds)
                           Default: 60 seconds (1 minute)
        """
        self.check_interval = check_interval
        self.running = False
        self.thread = None
        self.app = None
        # Add state manager
        self.state_manager = SchedulerStateManager(
            scheduler_name='auto_refresh',
            description='Auto-refresh RSS feeds based on their refresh_interval',
            default_interval=check_interval
        )
        self.task_state = None

    def start(self, app):
        """Start the scheduler thread"""
        if self.running:
            logger.warning("Auto-refresh scheduler is already running")
            return

        self.app = app

        # Initialize state from database (with error handling)
        try:
            self.task_state = self.state_manager.initialize_state(app, self.check_interval)
            if not self.task_state:
                logger.warning("Failed to initialize scheduler state, running without persistence")

            # Check if should execute immediately
            if self.task_state and self.state_manager.should_execute_immediately(self.task_state):
                logger.info("Auto-refresh scheduler is overdue, executing immediately on startup")
                # Execute in background thread to avoid blocking startup
                immediate_thread = threading.Thread(target=self._immediate_execution, daemon=True)
                immediate_thread.start()
        except Exception as e:
            logger.error(f"Error initializing scheduler state: {e}, continuing without persistence")
            self.task_state = None

        self.running = True
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()
        logger.info("Auto-refresh scheduler started")

    def _immediate_execution(self):
        """Execute immediately on startup if overdue"""
        try:
            with self.app.app_context():
                self._check_and_refresh_feeds()
        except Exception as e:
            logger.error(f"Error in immediate execution: {str(e)}", exc_info=True)

    def stop(self):
        """Stop the scheduler thread"""
        self.running = False
        if self.thread:
            self.thread.join(timeout=5)
        logger.info("Auto-refresh scheduler stopped")

    def _run(self):
        """Main scheduler loop"""
        logger.info("Auto-refresh scheduler loop started")

        while self.running:
            start_time = time.time()

            try:
                if not self.app:
                    logger.error("App context not available, skipping cycle")
                    time.sleep(self.check_interval)
                    continue

                with self.app.app_context():
                    try:
                        # Reload state from database at the start of each cycle
                        self.task_state = SchedulerTask.query.filter_by(
                            name='auto_refresh'
                        ).first()

                        # Pre-execution state update
                        if self.task_state:
                            self.state_manager.pre_execute(self.task_state)

                        # Execute the refresh check
                        self._check_and_refresh_feeds()

                        # Post-execution success update
                        duration = time.time() - start_time
                        if self.task_state:
                            # Get feed count within the same app context
                            try:
                                feed_count = Feed.query.filter(Feed.refresh_interval > 0).count()
                                result_msg = f"Checked {feed_count} feeds"
                            except Exception as e:
                                logger.warning(f"Failed to get feed count: {e}", exc_info=True)
                                result_msg = "Completed refresh check"

                            self.state_manager.post_execute_success(
                                self.task_state,
                                duration=duration,
                                result=result_msg
                            )
                    finally:
                        # 确保数据库会话被正确关闭，释放连接
                        db.session.remove()

            except Exception as e:
                logger.error(f"Error in auto-refresh scheduler: {str(e)}", exc_info=True)
                # Post-execution error update
                try:
                    with self.app.app_context():
                        try:
                            self.task_state = SchedulerTask.query.filter_by(
                                name='auto_refresh'
                            ).first()
                            if self.task_state:
                                self.state_manager.post_execute_error(self.task_state, str(e))
                        finally:
                            # 确保数据库会话被正确关闭
                            db.session.remove()
                except Exception as db_error:
                    logger.error(f"Failed to update error state: {db_error}")

            # Sleep for check_interval
            time.sleep(self.check_interval)

    def _check_and_refresh_feeds(self):
        """
        Check all feeds and refresh those that need it
        A feed needs refresh if:
        - refresh_interval > 0 (not set to manual only)
        - last_refresh_at is None (never refreshed)
        - OR (now - last_refresh_at) >= refresh_interval
        """
        try:
            feeds = Feed.query.filter(Feed.refresh_interval > 0).all()
            logger.debug(f"Checking {len(feeds)} feed(s) with refresh_interval > 0")

            if not feeds:
                logger.debug("No feeds with refresh_interval > 0 found")
                return

            now = datetime.utcnow()
            refresh_count = 0

            for feed in feeds:
                try:
                    # Check if feed needs refresh
                    needs_refresh = False

                    if feed.last_refresh_at is None:
                        # Never refreshed, do it now
                        needs_refresh = True
                        logger.info(f"Feed {feed.id} ({feed.title}) never refreshed, adding to queue")
                    else:
                        # Calculate time since last refresh
                        time_since_refresh = (now - feed.last_refresh_at).total_seconds()
                        interval_seconds = feed.refresh_interval * 60  # Convert minutes to seconds

                        logger.debug(f"Feed {feed.id}: time_since_refresh={int(time_since_refresh/60)} min, "
                                   f"interval={feed.refresh_interval} min, status={feed.refresh_status}")

                        if time_since_refresh >= interval_seconds:
                            needs_refresh = True
                            logger.info(f"Feed {feed.id} ({feed.title}) due for refresh " +
                                      f"(last refresh {int(time_since_refresh/60)} min ago, " +
                                      f"interval {feed.refresh_interval} min)")
                        else:
                            logger.debug(f"Feed {feed.id} ({feed.title}) not due yet "
                                       f"({int(time_since_refresh/60)} min < {feed.refresh_interval} min)")

                    # Refresh if needed, but skip if already refreshing
                    if needs_refresh and feed.refresh_status != 'refreshing':
                        # Update status and trigger background refresh
                        # Note: Don't update last_refresh_at here - let refresh_feed_worker do it
                        feed.refresh_status = 'refreshing'
                        feed.refresh_error = None
                        db.session.commit()

                        # Start refresh in background thread
                        logger.info(f"Starting background refresh for feed {feed.id} ({feed.title})")
                        thread = threading.Thread(
                            target=refresh_feed_worker,
                            args=(feed.id, self.app)
                        )
                        thread.daemon = True
                        thread.start()

                        refresh_count += 1
                    elif feed.refresh_status == 'refreshing':
                        logger.debug(f"Feed {feed.id} ({feed.title}) is already refreshing, skipping")

                except Exception as e:
                    logger.error(f"Error checking feed {feed.id}: {str(e)}", exc_info=True)
                    continue

            if refresh_count > 0:
                logger.info(f"Auto-refresh: triggered {refresh_count} feed(s) for refresh")
            else:
                logger.debug("No feeds needed refresh at this time")

        except Exception as e:
            logger.error(f"Error in _check_and_refresh_feeds: {str(e)}", exc_info=True)

    def get_status(self):
        """Get scheduler status"""
        base_status = {
            'running': self.running,
            'check_interval': self.check_interval,
            'thread_alive': self.thread.is_alive() if self.thread else False
        }

        # Add database state if available
        if self.app and self.state_manager:
            try:
                with self.app.app_context():
                    db_status = self.state_manager.get_status(self.app)
                    base_status['database_state'] = db_status
            except Exception as e:
                logger.error(f"Failed to get database state: {e}", exc_info=True)
                base_status['database_state'] = {
                    'status': 'error',
                    'error': str(e)
                }

        return base_status


# Global scheduler instance
scheduler = None


def init_scheduler(app):
    """Initialize and start the global scheduler"""
    global scheduler
    if scheduler is None:
        # Check interval can be configured via environment variable
        # Default: 60 seconds (check every minute)
        check_interval = int(app.config.get('AUTO_REFRESH_CHECK_INTERVAL', 60))
        scheduler = AutoRefreshScheduler(check_interval=check_interval)
        scheduler.start(app)
        logger.info(f"Auto-refresh scheduler initialized with {check_interval}s check interval")


def get_scheduler():
    """Get the global scheduler instance"""
    return scheduler


def stop_scheduler():
    """Stop the global scheduler"""
    global scheduler
    if scheduler:
        scheduler.stop()
        scheduler = None
