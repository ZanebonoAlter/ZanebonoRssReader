"""
Initialize database with sample data
"""
import requests
import json

API_BASE = 'http://127.0.0.1:5000/api'

# Sample categories
sample_categories = [
    {
        'name': '技术',
        'icon': 'mdi:code-tags',
        'color': '#3b82f6',
        'description': '技术开发、编程、架构'
    },
    {
        'name': '新闻',
        'icon': 'mdi:newspaper',
        'color': '#ef4444',
        'description': '时事新闻、热点追踪'
    },
    {
        'name': '设计',
        'icon': 'mdi:palette',
        'color': '#8b5cf6',
        'description': 'UI/UX、平面设计、创意'
    },
    {
        'name': '博客',
        'icon': 'mdi:post',
        'color': '#10b981',
        'description': '个人博客、随笔'
    },
    {
        'name': '人工智能',
        'icon': 'mdi:brain',
        'color': '#f59e0b',
        'description': 'AI、机器学习、深度学习'
    },
    {
        'name': '产品',
        'icon': 'mdi:cube-outline',
        'color': '#ec4899',
        'description': '产品设计、产品管理'
    }
]

# Sample feeds
sample_feeds = [
    {
        'url': 'https://www.ruanyifeng.com/blog/atom.xml',
        'category_name': '技术',
        'title': '阮一峰的网络日志',
        'icon': 'mdi:rss',
        'color': '#3b82f6'
    },
    {
        'url': 'https://sspai.com/feed',
        'category_name': '技术',
        'title': '少数派',
        'icon': 'mdi:tablet-ipad',
        'color': '#3b82f6'
    },
    {
        'url': 'https://36kr.com/feed',
        'category_name': '新闻',
        'title': '36氪',
        'icon': 'mdi:alpha-k',
        'color': '#ef4444'
    },
    {
        'url': 'https://www.infoq.cn/feed',
        'category_name': '技术',
        'title': 'InfoQ',
        'icon': 'mdi:alpha-q',
        'color': '#3b82f6'
    }
]

def init_categories():
    """Initialize categories"""
    print("Creating categories...")

    for category in sample_categories:
        response = requests.post(f'{API_BASE}/categories', json=category)
        if response.status_code in [200, 201]:
            print(f"  [OK] Created category: {category['name']}")
        elif response.status_code == 409:
            print(f"  [SKIP] Category already exists: {category['name']}")

def init_feeds():
    """Initialize feeds"""
    print("\nCreating feeds...")

    # Get category mapping
    response = requests.get(f'{API_BASE}/categories')
    categories = response.json()['data']
    category_map = {cat['name']: cat['id'] for cat in categories}

    for feed in sample_feeds:
        category_id = category_map.get(feed['category_name'])

        if not category_id:
            print(f"  [ERROR] Category not found: {feed['category_name']}")
            continue

        feed_data = {
            'url': feed['url'],
            'category_id': category_id,
            'title': feed['title'],
            'icon': feed['icon'],
            'color': feed['color']
        }

        response = requests.post(f'{API_BASE}/feeds', json=feed_data)
        if response.status_code in [200, 201]:
            result = response.json()
            print(f"  [OK] Created feed: {feed['title']}")
            if 'message' in result:
                print(f"       {result['message']}")
        elif response.status_code == 409:
            print(f"  [SKIP] Feed already exists: {feed['title']}")

if __name__ == '__main__':
    print("=" * 50)
    print("RSS Reader - Initialize Sample Data")
    print("=" * 50)

    try:
        init_categories()
        init_feeds()

        print("\n" + "=" * 50)
        print("[OK] Initialization complete!")
        print("=" * 50)

    except requests.exceptions.ConnectionError:
        print("\n[ERROR] Could not connect to backend server.")
        print("  Please make sure the backend is running on http://127.0.0.1:5000")
    except Exception as e:
        print(f"\n[ERROR] {str(e)}")
