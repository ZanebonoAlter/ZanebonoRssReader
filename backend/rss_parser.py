"""
RSS Parser with fallback for Python 3.13 compatibility
Uses feedparser 6.0.11+ which removed cgi dependency
"""
import feedparser
from datetime import datetime
from typing import Optional, Dict, List, Any


def parse_feed(url: str) -> Optional[Dict[str, Any]]:
    """
    Parse RSS/Atom feed from URL

    Args:
        url: RSS feed URL

    Returns:
        Dict with feed info and articles, or None if failed
    """
    try:
        # User agent to avoid blocking
        feedparser.USER_AGENT = 'RSS Reader/1.0 +https://github.com'

        # Parse feed
        feed = feedparser.parse(url)

        if feed.bozo and feed.bozo_exception:
            # Feed has errors, but might still be parseable
            print(f"Feed warning: {feed.bozo_exception}")

        # Extract feed info
        feed_info = {
            'title': feed.feed.get('title', 'Unknown Feed'),
            'description': feed.feed.get('description', ''),
            'link': feed.feed.get('link', ''),
            'image': feed.feed.get('image', {}).get('href') if feed.feed.get('image') else None,
            'language': feed.feed.get('language', 'en'),
        }

        # Extract articles (limit to 50 most recent)
        articles = []
        for entry in feed.entries[:50]:
            article = {
                'title': entry.get('title', 'No Title'),
                'link': entry.get('link', ''),
                'description': entry.get('description', ''),
                'content': extract_content(entry),
                'pub_date': parse_date(entry),
                'author': entry.get('author', ''),
                'tags': extract_tags(entry),
                'image_url': extract_image(entry),
            }
            articles.append(article)

        return {
            'feed': feed_info,
            'articles': articles,
            'total': len(articles)
        }

    except Exception as e:
        print(f"Error parsing feed {url}: {str(e)}")
        return None


def extract_content(entry: Dict[str, Any]) -> str:
    """Extract article content from various fields"""
    # Try content field
    if 'content' in entry and entry.content:
        content_list = entry.content if isinstance(entry.content, list) else [entry.content]
        if content_list and len(content_list) > 0:
            return content_list[0].get('value', '')

    # Try summary
    if 'summary' in entry:
        return entry.summary

    # Try description
    if 'description' in entry:
        return entry.description

    return ''


def parse_date(entry: Dict[str, Any]) -> Optional[str]:
    """Parse publication date from entry"""
    date_fields = ['published_parsed', 'updated_parsed']

    for field in date_fields:
        if field in entry and entry[field]:
            try:
                time_struct = entry[field]
                dt = datetime(*time_struct[:6])
                return dt.isoformat()
            except (TypeError, ValueError):
                continue

    # Fallback to string date
    date_str = entry.get('published') or entry.get('updated') or entry.get('date')
    if date_str:
        return date_str

    return datetime.now().isoformat()


def extract_tags(entry: Dict[str, Any]) -> List[str]:
    """Extract tags/categories from entry"""
    tags = []

    # Try tags field
    if 'tags' in entry and entry.tags:
        for tag in entry.tags[:10]:  # Limit to 10 tags
            if isinstance(tag, dict):
                term = tag.get('term')
                if term:
                    tags.append(term)
            elif isinstance(tag, str):
                tags.append(tag)

    # Try categories
    if 'categories' in entry and entry.categories:
        tags.extend(entry.categories[:5])

    return list(set(tags))  # Remove duplicates


def extract_image(entry: Dict[str, Any]) -> Optional[str]:
    """Extract main image from entry"""
    # Try enclosures (media files)
    if 'enclosures' in entry and entry.enclosures:
        for enclosure in entry.enclosures:
            if enclosure.get('type', '').startswith('image/'):
                return enclosure.get('href')

    # Try media_content
    if 'media_content' in entry and entry.media_content:
        media = entry.media_content[0] if isinstance(entry.media_content, list) else entry.media_content
        if media.get('type', '').startswith('image/'):
            return media.get('url')

    # Try to extract from content/summary HTML
    content = extract_content(entry)
    if content:
        import re
        img_match = re.search(r'<img[^>]+src="([^">]+)"', content)
        if img_match:
            return img_match.group(1)

    return None


def validate_feed_url(url: str) -> bool:
    """
    Validate if URL looks like a feed URL

    Args:
        url: URL to validate

    Returns:
        True if looks like a feed URL
    """
    url_lower = url.lower()

    # Check for common feed patterns
    feed_patterns = [
        '/feed',
        '/rss',
        '/atom',
        'rss.xml',
        'feed.xml',
        'atom.xml',
        '.rss',
        '.atom',
    ]

    return any(pattern in url_lower for pattern in feed_patterns)


if __name__ == '__main__':
    # Test with a real feed
    test_url = 'https://www.ruanyifeng.com/blog/atom.xml'
    result = parse_feed(test_url)

    if result:
        print(f"Feed: {result['feed']['title']}")
        print(f"Description: {result['feed']['description']}")
        print(f"Total articles: {result['total']}")
        print(f"\nFirst article:")
        if result['articles']:
            article = result['articles'][0]
            print(f"  Title: {article['title']}")
            print(f"  Date: {article['pub_date']}")
            print(f"  Link: {article['link']}")
