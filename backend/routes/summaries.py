"""
AI Summary Routes for category/article aggregation summaries
"""
from flask import Blueprint, request, jsonify
from models import AISummary, Article, Category, Feed, db
from datetime import datetime, timedelta, timezone
from routes.ai import AISummaryService
import json
import logging
import threading

summaries_bp = Blueprint('summaries', __name__, url_prefix='/api/summaries')
logger = logging.getLogger(__name__)

# Shanghai timezone (UTC+8)
SHANGHAI_TZ = timezone(timedelta(hours=8))

def get_current_cst_time():
    """Get current time in Shanghai timezone"""
    return datetime.now(SHANGHAI_TZ)


def get_ai_settings():
    """Get AI settings from request or use defaults"""
    data = request.get_json() if request.is_json else {}
    return {
        'base_url': data.get('base_url', 'https://api.openai.com/v1'),
        'api_key': data.get('api_key', ''),
        'model': data.get('model', 'gpt-4o-mini')
    }


@summaries_bp.route('', methods=['GET'])
def get_summaries():
    """Get all AI summaries, optionally filtered by category"""
    try:
        category_id = request.args.get('category_id', type=int)
        page = request.args.get('page', 1, type=int)
        per_page = request.args.get('per_page', 20, type=int)

        query = AISummary.query

        if category_id is not None:
            query = query.filter_by(category_id=category_id)

        # Order by created_at desc
        query = query.order_by(AISummary.created_at.desc())

        # Pagination
        pagination = query.paginate(
            page=page, per_page=per_page, error_out=False
        )

        return jsonify({
            'success': True,
            'data': [summary.to_dict() for summary in pagination.items],
            'pagination': {
                'page': page,
                'per_page': per_page,
                'total': pagination.total,
                'pages': pagination.pages
            }
        }), 200

    except Exception as e:
        logger.error(f"Error getting summaries: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@summaries_bp.route('/<int:summary_id>', methods=['GET'])
def get_summary(summary_id):
    """Get a specific AI summary"""
    try:
        summary = AISummary.query.get(summary_id)
        if not summary:
            return jsonify({
                'success': False,
                'error': 'Summary not found'
            }), 404

        # Parse article IDs to get full article details
        article_ids = json.loads(summary.articles) if summary.articles else []
        articles = Article.query.filter(Article.id.in_(article_ids)).all()

        summary_dict = summary.to_dict()
        summary_dict['article_details'] = [article.to_dict() for article in articles]

        return jsonify({
            'success': True,
            'data': summary_dict
        }), 200

    except Exception as e:
        logger.error(f"Error getting summary: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@summaries_bp.route('/generate', methods=['POST'])
def generate_summary():
    """
    Generate an AI summary for articles in a category within a time range

    Request body:
    {
        "category_id": 1,  // null for all categories
        "time_range": 180,  // minutes (default 180 = 3 hours)
        "base_url": "https://api.openai.com/v1",
        "api_key": "sk-...",
        "model": "gpt-4o-mini"
    }
    """
    try:
        data = request.get_json()

        # Validate required fields
        if not data.get('api_key'):
            return jsonify({
                'success': False,
                'error': 'API key is required'
            }), 400

        category_id = data.get('category_id')
        time_range = data.get('time_range', 180)  # Default 3 hours

        # Calculate time threshold
        time_threshold = datetime.utcnow() - timedelta(minutes=time_range)

        # Get articles based on category filter
        if category_id:
            # Get feeds in this category with AI summary enabled
            feeds = Feed.query.filter_by(
                category_id=category_id,
                ai_summary_enabled=True
            ).all()
            feed_ids = [feed.id for feed in feeds]

            # Get articles from these feeds
            articles = Article.query.filter(
                Article.feed_id.in_(feed_ids),
                Article.pub_date >= time_threshold
            ).order_by(Article.pub_date.desc()).all()

            category = Category.query.get(category_id)
            category_name = category.name if category else '未知分类'
        else:
            # Get all feeds with AI summary enabled
            feeds = Feed.query.filter_by(ai_summary_enabled=True).all()
            feed_ids = [feed.id for feed in feeds]

            # Get articles from enabled feeds only
            articles = Article.query.filter(
                Article.feed_id.in_(feed_ids),
                Article.pub_date >= time_threshold
            ).order_by(Article.pub_date.desc()).all()
            category_name = '全部分类'

        if not articles:
            return jsonify({
                'success': False,
                'error': f'在最近 {time_range} 分钟内没有找到文章'
            }), 404

        # Prepare article content for summarization
        article_texts = []
        for article in articles[:50]:  # Limit to 50 articles to avoid token limits
            text = f"标题: {article.title}\n"
            if article.description:
                text += f"描述: {article.description[:500]}\n"
            if article.content:
                text += f"内容: {article.content[:1000]}\n"
            text += f"链接: {article.link}\n"
            article_texts.append(text)

        articles_text = "\n---\n".join(article_texts)

        # Create AI service
        ai_service = AISummaryService(
            base_url=data.get('base_url', 'https://api.openai.com/v1'),
            api_key=data['api_key'],
            model=data.get('model', 'gpt-4o-mini')
        )

        # Generate summary using a specialized prompt
        summary_prompt = f"""请对以下来自"{category_name}"分类的 {len(articles)} 篇文章进行汇总总结。

文章列表（按时间倒序）：
{articles_text}

请提供以下格式的总结：

## 核心主题
用一句话概括这批文章的核心主题和趋势。

## 重要新闻

### 🔥 热点事件
列出2-3个最重要的事件，每个事件包含：
- 事件标题（用加粗）
- 简要说明（2-3句话）
- 引文标注新闻来源（使用 > [来源名称](链接) 格式）

### 📰 其他重要新闻
列出其他重要新闻，每条包含：
- 新闻标题（用加粗）
- 简要说明（1-2句话）
- 引文标注新闻来源（使用 > [来源名称](链接) 格式）

## 核心观点
总结3-5个核心观点或趋势，每个观点用简洁的语言表达。

## 相关标签
#标签1 #标签2 #标签3

**重要提醒**：
1. 必须为每条新闻标注来源，使用引文格式
2. 来源格式：> [来源订阅源名称](文章链接)
3. 确保总结简洁明了，突出重点
4. 保持客观中立的语气"""

        # Import requests here for the API call
        import requests

        response = requests.post(
            f'{ai_service.base_url}/chat/completions',
            headers=ai_service.headers,
            json={
                'model': ai_service.model,
                'messages': [
                    {'role': 'system', 'content': '你是一个专业的新闻分析助手，擅长汇总和分析多篇文章。'},
                    {'role': 'user', 'content': summary_prompt}
                ],
                'temperature': 0.7,
                'max_tokens': 3000
            },
            timeout=120
        )

        # Check for errors before raising
        try:
            response.raise_for_status()
        except requests.HTTPError as e:
            # Try to get detailed error message from response
            error_detail = str(e)
            try:
                error_json = response.json()
                error_detail = f"{error_detail}\nResponse: {json.dumps(error_json, indent=2, ensure_ascii=False)}"
            except:
                # If response is not JSON, try to get text
                error_detail = f"{error_detail}\nResponse text: {response.text[:500]}"
            logger.error(f"API request failed: {error_detail}")
            raise

        result = response.json()

        # Parse the response
        summary_text = result['choices'][0]['message']['content']

        # Generate title
        title = f"{category_name} - {get_current_cst_time().strftime('%Y-%m-%d %H:%M')} 新闻汇总"

        # Create AI summary record
        ai_summary = AISummary(
            category_id=category_id,
            title=title,
            summary=summary_text,
            articles=json.dumps([article.id for article in articles]),
            article_count=len(articles),
            time_range=time_range
        )

        db.session.add(ai_summary)
        db.session.commit()

        return jsonify({
            'success': True,
            'data': ai_summary.to_dict(),
            'message': f'成功生成 {len(articles)} 篇文章的汇总总结'
        }), 201

    except Exception as e:
        db.session.rollback()
        logger.error(f"Error generating summary: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@summaries_bp.route('/<int:summary_id>', methods=['DELETE'])
def delete_summary(summary_id):
    """Delete an AI summary"""
    try:
        summary = AISummary.query.get(summary_id)
        if not summary:
            return jsonify({
                'success': False,
                'error': 'Summary not found'
            }), 404

        db.session.delete(summary)
        db.session.commit()

        return jsonify({
            'success': True,
            'message': 'Summary deleted successfully'
        }), 200

    except Exception as e:
        db.session.rollback()
        logger.error(f"Error deleting summary: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@summaries_bp.route('/auto-generate', methods=['POST'])
def auto_generate_summary():
    """
    Generate summary asynchronously in background
    Same parameters as /generate but runs in background thread
    """
    try:
        data = request.get_json()

        # Start generation in background thread
        app = request.environ.get('flask.app')
        thread = threading.Thread(
            target=_generate_summary_worker,
            args=(data, app._get_current_object() if app else None)
        )
        thread.daemon = True
        thread.start()

        return jsonify({
            'success': True,
            'message': '后台生成任务已启动'
        }), 202

    except Exception as e:
        logger.error(f"Error starting auto generation: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


def _generate_summary_worker(data, app):
    """Worker function to generate summary in background"""
    if app:
        with app.app_context():
            try:
                category_id = data.get('category_id')
                time_range = data.get('time_range', 180)
                category_name = "全部分类" if category_id is None else f"分类 ID {category_id}"

                logger.info(f"[Worker] Starting summary generation for {category_name}, time_range={time_range} min")

                # Calculate time threshold
                time_threshold = datetime.utcnow() - timedelta(minutes=time_range)
                logger.debug(f"[Worker] Time threshold: {time_threshold} (UTC)")

                # Get articles
                if category_id:
                    # Get feeds in this category with AI summary enabled
                    logger.debug(f"[Worker] Querying feeds for category_id={category_id} with ai_summary_enabled=True")
                    feeds = Feed.query.filter_by(
                        category_id=category_id,
                        ai_summary_enabled=True
                    ).all()
                    feed_ids = [feed.id for feed in feeds]
                    logger.info(f"[Worker] Found {len(feeds)} feed(s) with AI summary enabled in category {category_id}")
                    
                    articles = Article.query.filter(
                        Article.feed_id.in_(feed_ids),
                        Article.pub_date >= time_threshold
                    ).order_by(Article.pub_date.desc()).all()
                    category = Category.query.get(category_id)
                    category_name = category.name if category else '未知分类'
                else:
                    # Get all feeds with AI summary enabled
                    logger.debug("[Worker] Querying all feeds with ai_summary_enabled=True")
                    feeds = Feed.query.filter_by(ai_summary_enabled=True).all()
                    feed_ids = [feed.id for feed in feeds]
                    logger.info(f"[Worker] Found {len(feeds)} feed(s) with AI summary enabled (all categories)")
                    
                    articles = Article.query.filter(
                        Article.feed_id.in_(feed_ids),
                        Article.pub_date >= time_threshold
                    ).order_by(Article.pub_date.desc()).all()
                    category_name = '全部分类'

                logger.info(f"[Worker] Found {len(articles)} article(s) in time range for {category_name}")

                if not articles:
                    logger.info(f"[Worker] No articles found for auto-summary in category {category_id}, skipping")
                    return

                # Prepare content
                logger.debug(f"[Worker] Preparing content from {min(len(articles), 50)} article(s)")
                article_texts = []
                for idx, article in enumerate(articles[:50], 1):
                    text = f"标题: {article.title}\n"
                    if article.description:
                        text += f"描述: {article.description[:500]}\n"
                    if article.content:
                        text += f"内容: {article.content[:1000]}\n"
                    text += f"链接: {article.link}\n"
                    article_texts.append(text)
                    if idx % 10 == 0:
                        logger.debug(f"[Worker] Processed {idx}/{min(len(articles), 50)} articles")

                articles_text = "\n---\n".join(article_texts)
                logger.debug(f"[Worker] Prepared article text, total length: {len(articles_text)} characters")

                # Create AI service
                logger.debug(f"[Worker] Creating AI service: model={data.get('model', 'gpt-4o-mini')}, base_url={data.get('base_url', 'https://api.openai.com/v1')}")
                ai_service = AISummaryService(
                    base_url=data.get('base_url', 'https://api.openai.com/v1'),
                    api_key=data['api_key'],
                    model=data.get('model', 'gpt-4o-mini')
                )

                import requests

                logger.info(f"[Worker] Sending request to AI API for {category_name} ({len(articles)} articles)")
                summary_prompt = f"""请对以下来自"{category_name}"分类的 {len(articles)} 篇文章进行汇总总结。

文章列表（按时间倒序）：
{articles_text}

请提供以下格式的总结：

## 核心主题
用一句话概括这批文章的核心主题和趋势。

## 重要新闻

### 🔥 热点事件
列出2-3个最重要的事件，每个事件包含：
- 事件标题（用加粗）
- 简要说明（2-3句话）
- 引文标注新闻来源（使用 > [来源名称](链接) 格式）

### 📰 其他重要新闻
列出其他重要新闻，每条包含：
- 新闻标题（用加粗）
- 简要说明（1-2句话）
- 引文标注新闻来源（使用 > [来源名称](链接) 格式）

## 核心观点
总结3-5个核心观点或趋势，每个观点用简洁的语言表达。

## 相关标签
#标签1 #标签2 #标签3

**重要提醒**：
1. 必须为每条新闻标注来源，使用引文格式
2. 来源格式：> [来源订阅源名称](文章链接)
3. 确保总结简洁明了，突出重点
4. 保持客观中立的语气"""

                logger.debug(f"[Worker] API request: POST {ai_service.base_url}/chat/completions")
                logger.debug(f"[Worker] Request params: model={ai_service.model}, temperature=0.7, max_tokens=3000")
                
                response = requests.post(
                    f'{ai_service.base_url}/chat/completions',
                    headers=ai_service.headers,
                    json={
                        'model': ai_service.model,
                        'messages': [
                            {'role': 'system', 'content': '你是一个专业的新闻分析助手，擅长汇总和分析多篇文章。'},
                            {'role': 'user', 'content': summary_prompt}
                        ],
                        'temperature': 0.7,
                        'max_tokens': 3000
                    },
                    timeout=120
                )
                
                logger.debug(f"[Worker] API response status: {response.status_code}")

                # Check for errors before raising
                try:
                    response.raise_for_status()
                except requests.HTTPError as e:
                    # Try to get detailed error message from response
                    error_detail = str(e)
                    try:
                        error_json = response.json()
                        error_detail = f"{error_detail}\nResponse: {json.dumps(error_json, indent=2, ensure_ascii=False)}"
                    except:
                        # If response is not JSON, try to get text
                        error_detail = f"{error_detail}\nResponse text: {response.text[:500]}"
                    logger.error(f"API request failed in worker: {error_detail}")
                    raise

                result = response.json()
                summary_text = result['choices'][0]['message']['content']
                logger.info(f"[Worker] Received AI response, summary length: {len(summary_text)} characters")

                # Generate title
                title = f"{category_name} - {get_current_cst_time().strftime('%Y-%m-%d %H:%M')} 新闻汇总"
                logger.debug(f"[Worker] Generated title: {title}")

                # Create AI summary record
                logger.debug(f"[Worker] Creating AI summary record in database")
                ai_summary = AISummary(
                    category_id=category_id,
                    title=title,
                    summary=summary_text,
                    articles=json.dumps([article.id for article in articles]),
                    article_count=len(articles),
                    time_range=time_range
                )

                db.session.add(ai_summary)
                db.session.commit()
                logger.info(f"[Worker] Successfully saved summary to database with ID: {ai_summary.id}")

                logger.info(f"[Worker] Auto-generated summary for {category_name} (category_id: {category_id}): {len(articles)} articles")

            except Exception as e:
                logger.error(f"[Worker] Error in auto-summary worker for category {data.get('category_id')}: {e}", exc_info=True)
                db.session.rollback()
            finally:
                # 确保数据库会话被正确关闭，释放连接
                # 在多线程环境中，显式关闭会话可以防止连接泄漏
                db.session.remove()
    else:
        logger.error("No Flask app context available for auto-summary worker")
