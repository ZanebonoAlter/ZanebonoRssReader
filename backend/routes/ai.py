"""
AI Summary Routes
Provides AI-powered article summarization using OpenAI-compatible APIs
"""
from flask import Blueprint, request, jsonify
import requests
import logging
import json
from functools import lru_cache
from models import AISettings, db

ai_bp = Blueprint('ai', __name__, url_prefix='/api/ai')
logger = logging.getLogger(__name__)


class AISummaryService:
    """Service for AI article summarization"""

    def __init__(self, base_url: str, api_key: str, model: str):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.model = model
        self.headers = {
            'Authorization': f'Bearer {api_key}',
            'Content-Type': 'application/json'
        }

    def summarize_article(self, title: str, content: str, language: str = 'zh') -> dict:
        """
        Summarize an article using AI

        Args:
            title: Article title
            content: Article content
            language: Target language for summary ('zh' for Chinese, 'en' for English)

        Returns:
            Dictionary with summary, key points, and tags
        """
        try:
            # Prepare the prompt
            system_prompt = self._get_system_prompt(language)
            user_content = self._prepare_article_content(title, content)

            # Make API request
            response = requests.post(
                f'{self.base_url}/chat/completions',
                headers=self.headers,
                json={
                    'model': self.model,
                    'messages': [
                        {'role': 'system', 'content': system_prompt},
                        {'role': 'user', 'content': user_content}
                    ],
                    'temperature': 0.7,
                    'max_tokens': 2000
                },
                timeout=60
            )

            response.raise_for_status()
            result = response.json()

            # Parse the response
            summary_text = result['choices'][0]['message']['content']

            # Try to parse structured response
            return self._parse_summary_response(summary_text)

        except requests.exceptions.RequestException as e:
            logger.error(f"AI API request failed: {e}")
            raise Exception(f"AI API 请求失败: {str(e)}")
        except Exception as e:
            logger.error(f"Summary generation failed: {e}")
            raise Exception(f"生成总结失败: {str(e)}")

    def _get_system_prompt(self, language: str) -> str:
        """Get system prompt for the AI model"""
        if language == 'zh':
            return """你是一个专业的文章分析助手。请对给定的文章进行智能总结，回复格式如下：

## 一句话总结
用一句话概括文章的核心内容。

## 核心观点
- 观点1
- 观点2
- 观点3

## 关键要点
1. 要点一
2. 要点二
3. 要点三

## 标签
#标签1 #标签2 #标签3

请确保总结简洁明了，突出重点。"""
        else:
            return """You are a professional article analysis assistant. Please provide an intelligent summary of the given article in the following format:

## One-Sentence Summary
A single sentence summarizing the core content of the article.

## Key Points
- Point 1
- Point 2
- Point 3

## Main Takeaways
1. Takeaway 1
2. Takeaway 2
3. Takeaway 3

## Tags
#tag1 #tag2 #tag3

Please ensure the summary is concise and highlights the key points."""

    def _prepare_article_content(self, title: str, content: str) -> str:
        """Prepare article content for the AI model"""
        # Limit content length to avoid token limits
        max_content_length = 8000
        if len(content) > max_content_length:
            content = content[:max_content_length] + '...'

        return f"标题：{title}\n\n内容：{content}"

    def _parse_summary_response(self, response_text: str) -> dict:
        """Parse the AI response into structured data"""
        summary = {
            'one_sentence': '',
            'key_points': [],
            'takeaways': [],
            'tags': []
        }

        current_section = None
        lines = response_text.split('\n')

        for line in lines:
            line = line.strip()

            if not line:
                continue

            # Detect sections
            if '一句话总结' in line or 'One-Sentence Summary' in line:
                current_section = 'one_sentence'
                continue
            elif '核心观点' in line or 'Key Points' in line:
                current_section = 'key_points'
                continue
            elif '关键要点' in line or 'Main Takeaways' in line:
                current_section = 'takeaways'
                continue
            elif '标签' in line or 'Tags' in line:
                current_section = 'tags'
                continue

            # Parse content based on current section
            if current_section == 'one_sentence' and line:
                summary['one_sentence'] = line.lstrip('•-*').strip()
            elif current_section == 'key_points':
                point = line.lstrip('•-*').strip()
                if point:
                    summary['key_points'].append(point)
            elif current_section == 'takeaways':
                takeaway = line.lstrip('•-*123456789.).').strip()
                if takeaway:
                    summary['takeaways'].append(takeaway)
            elif current_section == 'tags':
                tags = [tag.strip('#').strip() for tag in line.split() if tag.startswith('#')]
                summary['tags'].extend(tags)

        # If parsing failed, provide a fallback
        if not summary['one_sentence'] and response_text:
            summary['one_sentence'] = response_text[:200]

        return summary


@lru_cache(maxsize=1)
def get_ai_service():
    """Get or create AI service instance (cached)"""
    # For now, return None - service will be initialized with credentials from request
    return None


@ai_bp.route('/summarize', methods=['POST'])
def summarize_article():
    """
    Summarize an article using AI

    Request body:
    {
        "base_url": "https://api.openai.com/v1",
        "api_key": "sk-...",
        "model": "gpt-4o-mini",
        "title": "Article Title",
        "content": "Article content...",
        "language": "zh"
    }

    Response:
    {
        "success": true,
        "data": {
            "one_sentence": "One sentence summary",
            "key_points": ["point1", "point2"],
            "takeaways": ["takeaway1", "takeaway2"],
            "tags": ["tag1", "tag2"]
        }
    }
    """
    try:
        data = request.get_json()

        # Validate required fields
        required_fields = ['base_url', 'api_key', 'model', 'title', 'content']
        for field in required_fields:
            if field not in data:
                return jsonify({
                    'success': False,
                    'error': f'Missing required field: {field}'
                }), 400

        # Create AI service instance
        ai_service = AISummaryService(
            base_url=data['base_url'],
            api_key=data['api_key'],
            model=data['model']
        )

        # Get language (default to Chinese)
        language = data.get('language', 'zh')

        # Generate summary
        summary = ai_service.summarize_article(
            title=data['title'],
            content=data['content'],
            language=language
        )

        return jsonify({
            'success': True,
            'data': summary
        }), 200

    except Exception as e:
        logger.error(f"Error in summarize_article: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@ai_bp.route('/test', methods=['POST'])
def test_connection():
    """
    Test AI API connection

    Request body:
    {
        "base_url": "https://api.openai.com/v1",
        "api_key": "sk-...",
        "model": "gpt-4o-mini"
    }
    """
    try:
        data = request.get_json()

        # Validate required fields
        required_fields = ['base_url', 'api_key', 'model']
        for field in required_fields:
            if field not in data:
                return jsonify({
                    'success': False,
                    'error': f'Missing required field: {field}'
                }), 400

        # Create AI service instance and test
        ai_service = AISummaryService(
            base_url=data['base_url'],
            api_key=data['api_key'],
            model=data['model']
        )

        # Make a simple test request
        response = requests.post(
            f'{ai_service.base_url}/chat/completions',
            headers=ai_service.headers,
            json={
                'model': ai_service.model,
                'messages': [
                    {'role': 'user', 'content': 'Hi'}
                ],
                'max_tokens': 10
            },
            timeout=30
        )

        response.raise_for_status()

        return jsonify({
            'success': True,
            'message': '连接测试成功'
        }), 200

    except requests.exceptions.RequestException as e:
        logger.error(f"AI connection test failed: {e}")
        return jsonify({
            'success': False,
            'error': f'连接测试失败: {str(e)}'
        }), 400
    except Exception as e:
        logger.error(f"Error in test_connection: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@ai_bp.route('/settings', methods=['GET'])
def get_ai_settings():
    """
    Get AI settings from database

    Response:
    {
        "success": true,
        "data": {
            "base_url": "https://api.openai.com/v1",
            "api_key": "sk-...",
            "model": "gpt-4o-mini"
        }
    }
    """
    try:
        settings = AISettings.query.filter_by(key='summary_config').first()

        if settings and settings.value:
            config = json.loads(settings.value)
            # Don't expose the full API key in the response
            config_with_masked_key = {
                **config,
                'api_key': config.get('api_key', '')[:10] + '...' if config.get('api_key') else ''
            } if config.get('api_key') else config

            return jsonify({
                'success': True,
                'data': config_with_masked_key
            }), 200
        else:
            return jsonify({
                'success': True,
                'data': None
            }), 200

    except Exception as e:
        logger.error(f"Error getting AI settings: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@ai_bp.route('/settings', methods=['POST'])
def save_ai_settings():
    """
    Save AI settings to database

    Request body:
    {
        "base_url": "https://api.openai.com/v1",
        "api_key": "sk-...",
        "model": "gpt-4o-mini"
    }

    Response:
    {
        "success": true,
        "message": "AI settings saved successfully"
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

        config = {
            'base_url': data.get('base_url', 'https://api.openai.com/v1'),
            'api_key': data['api_key'],
            'model': data.get('model', 'gpt-4o-mini')
        }

        # Save or update settings in database
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

        # Also update the running scheduler if it exists
        try:
            from auto_summary import get_auto_summary_scheduler
            scheduler = get_auto_summary_scheduler()
            if scheduler:
                scheduler.set_ai_config(
                    base_url=config['base_url'],
                    api_key=config['api_key'],
                    model=config['model']
                )
                logger.info("Updated running scheduler with new AI configuration")
        except ImportError:
            logger.warning("Could not import auto_summary scheduler")
        except Exception as scheduler_error:
            logger.error(f"Error updating scheduler: {scheduler_error}")

        logger.info("AI settings saved to database")

        return jsonify({
            'success': True,
            'message': 'AI settings saved successfully'
        }), 200

    except Exception as e:
        db.session.rollback()
        logger.error(f"Error saving AI settings: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500
