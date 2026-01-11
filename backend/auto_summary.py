"""Auto-summary scheduler for AI-powered article aggregation summaries"""
import threading
import time
import logging
import json
from datetime import datetime
from models import Category, Feed, AISummary, db, SchedulerTask, AISettings
from routes.summaries import _generate_summary_worker
from flask import current_app
from scheduler_base import SchedulerStateManager

logger = logging.getLogger(__name__)
# Set logger level to DEBUG to see detailed logs
logger.setLevel(logging.DEBUG)


class AutoSummaryScheduler:
    """Background scheduler to auto-generate AI summaries for categories"""

    def __init__(self, check_interval=3600):
        """
        Initialize the auto-summary scheduler

        Args:
            check_interval: How often to generate summaries (seconds)
                           Default: 3600 seconds (1 hour)
        """
        self.check_interval = check_interval
        self.running = False
        self.thread = None
        self.app = None
        self.ai_config = None
        # Add state manager
        self.state_manager = SchedulerStateManager(
            scheduler_name='auto_summary',
            description='Auto-generate AI summaries for categories',
            default_interval=check_interval
        )
        self.task_state = None
        # Add execution lock to prevent concurrent summary generation
        self._execution_lock = threading.Lock()
        self._is_executing = False

    def set_ai_config(self, base_url, api_key, model):
        """Set AI configuration for summary generation and persist to database"""
        config = {
            'base_url': base_url,
            'api_key': api_key,
            'model': model
        }
        self.ai_config = config

        # Persist to database
        try:
            with db.session.begin_nested():
                settings = AISettings.query.filter_by(key='summary_config').first()
                config_json = json.dumps(config)
                if settings:
                    settings.value = config_json
                else:
                    settings = AISettings(
                        key='summary_config',
                        value=config_json,
                        description='AI summary generation configuration'
                    )
                    db.session.add(settings)
            db.session.commit()
            logger.info("AI configuration saved to database")
        except Exception as e:
            logger.error(f"Failed to save AI config to database: {e}")
            db.session.rollback()

    def load_ai_config_from_db(self):
        """Load AI configuration from database"""
        try:
            settings = AISettings.query.filter_by(key='summary_config').first()
            if settings and settings.value:
                config = json.loads(settings.value)
                self.ai_config = config
                logger.info("AI configuration loaded from database")
                return True
            return False
        except Exception as e:
            logger.error(f"Failed to load AI config from database: {e}")
            return False

    def start(self, app):
        """Start the scheduler thread"""
        if self.running:
            logger.warning("Auto-summary scheduler is already running")
            return

        self.app = app

        # Load AI configuration from database on startup
        try:
            with app.app_context():
                loaded = self.load_ai_config_from_db()
                if loaded:
                    logger.info("Loaded AI configuration from database on startup")
                else:
                    logger.info("No AI configuration found in database")
        except Exception as e:
            logger.error(f"Error loading AI config on startup: {e}")

        # Initialize state from database (with error handling)
        try:
            self.task_state = self.state_manager.initialize_state(app, self.check_interval)
            if not self.task_state:
                logger.warning("Failed to initialize scheduler state, running without persistence")

            # Check if should execute immediately
            if self.task_state and self.state_manager.should_execute_immediately(self.task_state):
                if self.ai_config:
                    logger.info("Auto-summary scheduler is overdue, executing immediately on startup")
                    # Execute in background thread to avoid blocking startup
                    immediate_thread = threading.Thread(target=self._immediate_execution, daemon=True)
                    immediate_thread.start()
                else:
                    logger.info("Auto-summary scheduler is overdue but AI config not set, skipping immediate execution")
        except Exception as e:
            logger.error(f"Error initializing scheduler state: {e}, continuing without persistence")
            self.task_state = None

        self.running = True
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()
        logger.info("Auto-summary scheduler started")

    def _immediate_execution(self):
        """Execute immediately on startup if overdue"""
        try:
            with self.app.app_context():
                self._generate_summaries_for_categories()
        except Exception as e:
            logger.error(f"Error in immediate execution: {str(e)}", exc_info=True)

    def stop(self):
        """Stop the scheduler thread"""
        self.running = False
        if self.thread:
            self.thread.join(timeout=5)
        logger.info("Auto-summary scheduler stopped")

    def _run(self):
        """Main scheduler loop"""
        logger.info("Auto-summary scheduler loop started")

        while self.running:
            start_time = time.time()

            try:
                with self.app.app_context():
                    # Reload state from database at the start of each cycle
                    self.task_state = SchedulerTask.query.filter_by(
                        name='auto_summary'
                    ).first()

                    # Check if AI config is set
                    if not self.ai_config:
                        logger.debug("AI config not set, skipping summary generation")
                        # Still update state to idle
                        if self.task_state:
                            self.task_state.status = 'idle'
                            db.session.commit()
                    else:
                        # Check if it's time to execute (check next_execution_time)
                        should_execute = True
                        if self.task_state and self.task_state.next_execution_time:
                            now = datetime.utcnow()
                            if now < self.task_state.next_execution_time:
                                should_execute = False
                                time_until_next = (self.task_state.next_execution_time - now).total_seconds()
                                logger.debug(f"Not time to execute yet, next execution in {time_until_next:.0f} seconds")

                        if should_execute:
                            # Pre-execution state update
                            if self.task_state:
                                self.state_manager.pre_execute(self.task_state)

                            logger.info("Starting auto-summary generation cycle")
                            self._generate_summaries_for_categories()
                            logger.info("Auto-summary generation cycle completed")

                            # Post-execution success update
                            duration = time.time() - start_time
                            if self.task_state:
                                category_count = Category.query.count()
                                self.state_manager.post_execute_success(
                                    self.task_state,
                                    duration=duration,
                                    result=f"Generated summaries for {category_count} categories"
                                )
                        else:
                            # Update status to idle even when not executing
                            if self.task_state:
                                self.task_state.status = 'idle'
                                db.session.commit()

            except Exception as e:
                logger.error(f"Error in auto-summary scheduler: {str(e)}", exc_info=True)
                # Post-execution error update
                try:
                    with self.app.app_context():
                        self.task_state = SchedulerTask.query.filter_by(
                            name='auto_summary'
                        ).first()
                        if self.task_state:
                            self.state_manager.post_execute_error(self.task_state, str(e))
                except Exception as db_error:
                    logger.error(f"Failed to update error state: {db_error}")

            # Sleep for check_interval
            logger.debug(f"Sleeping for {self.check_interval} seconds until next cycle")
            time.sleep(self.check_interval)

    def _generate_summaries_for_categories(self):
        """Generate summaries for all categories that have recent articles"""
        # Check if already executing (concurrency check)
        if not self._execution_lock.acquire(blocking=False):
            logger.warning("Summary generation already in progress, skipping this execution")
            return

        try:
            self._is_executing = True
            logger.info("Acquired execution lock, starting summary generation")

            # Get all categories
            categories = Category.query.all()
            logger.info(f"Found {len(categories)} categories to process")

            # Only process "all categories" summary (category_id = None)
            # This will cover all feeds with ai_summary_enabled=True
            # Note: We don't generate summaries for individual categories to avoid duplicate processing
            # 为每个 category 生成 summary（不再生成 "全部分类" summary）
            # Generate for each category
            for idx, category in enumerate(categories, 1):
                logger.debug(f"Processing category {idx}/{len(categories)}: {category.name} (ID: {category.id})")
                self._generate_summary_for_category(category.id)

            logger.info(f"Completed processing {len(categories)} category summaries")

        except Exception as e:
            logger.error(f"Error generating summaries: {str(e)}", exc_info=True)
        finally:
            self._is_executing = False
            self._execution_lock.release()
            logger.info("Released execution lock")

    def _generate_summary_for_category(self, category_id):
        """Generate summary for a specific category"""
        try:
            category_name = "全部分类" if category_id is None else f"分类 ID {category_id}"
            logger.info(f"Starting summary generation for {category_name}")
            
            # Prepare data for summary generation
            data = {
                'category_id': category_id,
                'time_range': 180,  # Default 3 hours
                'base_url': self.ai_config['base_url'],
                'api_key': self.ai_config['api_key'],
                'model': self.ai_config['model']
            }

            logger.debug(f"Summary generation config: time_range={data['time_range']} min, "
                        f"model={data['model']}, base_url={data['base_url']}")

            # Generate summary in background
            logger.debug(f"Starting background worker thread for {category_name}")
            _generate_summary_worker(data, self.app)

            logger.info(f"Auto-summary generation triggered for {category_name} (category_id: {category_id})")

        except Exception as e:
            logger.error(f"Error generating summary for category {category_id}: {str(e)}", exc_info=True)

    def get_status(self):
        """Get scheduler status"""
        base_status = {
            'running': self.running,
            'check_interval': self.check_interval,
            'thread_alive': self.thread.is_alive() if self.thread else False,
            'ai_configured': self.ai_config is not None,
            'is_executing': self._is_executing,
            'execution_locked': self._execution_lock.locked()
        }

        # Add database state if available
        if self.app and self.state_manager:
            db_status = self.state_manager.get_status(self.app)
            base_status['database_state'] = db_status

        return base_status

    def is_executing(self):
        """Check if summary generation is currently in progress"""
        return self._is_executing or self._execution_lock.locked()


# Global scheduler instance
auto_summary_scheduler = None


def init_auto_summary_scheduler(app):
    """Initialize and start the global auto-summary scheduler"""
    global auto_summary_scheduler
    if auto_summary_scheduler is None:
        # Check interval can be configured via environment variable
        # Default: 3600 seconds (1 hour)
        check_interval = int(app.config.get('AUTO_SUMMARY_INTERVAL', 3600))
        auto_summary_scheduler = AutoSummaryScheduler(check_interval=check_interval)
        auto_summary_scheduler.start(app)
        logger.info(f"Auto-summary scheduler initialized with {check_interval}s interval")


def get_auto_summary_scheduler():
    """Get the global scheduler instance"""
    return auto_summary_scheduler


def update_auto_summary_config(base_url, api_key, model):
    """Update AI configuration for the auto-summary scheduler"""
    global auto_summary_scheduler
    if auto_summary_scheduler:
        auto_summary_scheduler.set_ai_config(base_url, api_key, model)
        logger.info("Auto-summary scheduler AI configuration updated")


def stop_auto_summary_scheduler():
    """Stop the global scheduler"""
    global auto_summary_scheduler
    if auto_summary_scheduler:
        auto_summary_scheduler.stop()
        auto_summary_scheduler = None
