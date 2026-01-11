from flask import Blueprint, request, jsonify, Response
from models import Feed, Category, db
from datetime import datetime, timedelta, timezone
import feedparser
import xml.etree.ElementTree as ET
from xml.dom import minidom
from tasks import enqueue_feeds

opml_bp = Blueprint('opml', __name__, url_prefix='/api')

# Shanghai timezone (UTC+8)
SHANGHAI_TZ = timezone(timedelta(hours=8))

def get_current_cst_time():
    """Get current time in Shanghai timezone"""
    return datetime.now(SHANGHAI_TZ)

@opml_bp.route('/import-opml', methods=['POST'])
def import_opml():
    """Import feeds from OPML file"""
    try:
        print(f"Request files: {request.files}")
        print(f"Request form: {request.form}")
        print(f"Content-Type: {request.content_type}")

        if 'file' not in request.files:
            return jsonify({
                'success': False,
                'error': 'No file provided'
            }), 400

        file = request.files['file']
        if file.filename == '':
            return jsonify({
                'success': False,
                'error': 'No file selected'
            }), 400

        if not file.filename.endswith('.opml') and not file.filename.endswith('.xml'):
            return jsonify({
                'success': False,
                'error': 'Invalid file format. Please upload an OPML file.'
            }), 400

        # Parse OPML
        content = file.read().decode('utf-8')
        print(f"File content length: {len(content)}")

        root = ET.fromstring(content)

        feeds_added = []
        categories_added = 0
        errors = []

        # Find body element
        body = root.find('.//body')
        if body is None:
            return jsonify({
                'success': False,
                'error': 'Invalid OPML format: no body element found'
            }), 400

        # Process top-level outlines (categories) and their children (feeds)
        for category_outline in body.findall('outline'):
            category_name = category_outline.get('title') or category_outline.get('text') or 'Uncategorized'

            # Create or get category
            category = Category.query.filter_by(name=category_name).first()
            if not category:
                category = Category(
                    name=category_name,
                    slug=category_name.lower().replace(' ', '-').replace('/', '-'),
                    icon='folder',
                    color='#6366f1'
                )
                db.session.add(category)
                db.session.flush()
                categories_added += 1

            # Process feeds in this category
            for feed_outline in category_outline.findall('outline'):
                try:
                    xml_url = feed_outline.get('xmlUrl')
                    title = feed_outline.get('title') or feed_outline.get('text') or 'Untitled Feed'

                    if not xml_url:
                        continue

                    # Check if feed already exists
                    existing_feed = Feed.query.filter_by(url=xml_url).first()
                    if existing_feed:
                        continue

                    # Create feed with basic info from OPML (no parsing yet)
                    feed = Feed(
                        title=title,
                        description='',
                        url=xml_url,
                        category_id=category.id,
                        icon='rss',
                        color='#8b5cf6',
                        last_updated=datetime.utcnow()
                    )
                    db.session.add(feed)
                    db.session.flush()  # Flush to get the ID
                    feeds_added.append(feed.id)
                    print(f"  Added feed: {title} (ID: {feed.id})")

                except Exception as e:
                    errors.append(f"Error importing feed '{title}': {str(e)}")
                    continue

        db.session.commit()

        # Enqueue feeds for async metadata update
        if feeds_added:
            enqueue_count = enqueue_feeds(feeds_added)
            print(f"Enqueued {enqueue_count} feeds for background processing")

        return jsonify({
            'success': True,
            'data': {
                'feeds_added': len(feeds_added),
                'categories_added': categories_added,
                'errors': errors,
                'async_update': True
            },
            'message': f'Imported {len(feeds_added)} feeds and {categories_added} categories. Feed metadata will be updated in the background.'
        }), 200

    except Exception as e:
        db.session.rollback()
        return jsonify({
            'success': False,
            'error': f'Error parsing OPML file: {str(e)}'
        }), 500

@opml_bp.route('/export-opml', methods=['GET'])
def export_opml():
    """Export feeds to OPML format"""
    try:
        categories = Category.query.all()

        # Create OPML structure
        opml = ET.Element('opml')
        opml.set('version', '2.0')

        head = ET.SubElement(opml, 'head')
        ET.SubElement(head, 'title').text = 'RSS Feeds Export'
        ET.SubElement(head, 'dateCreated').text = get_current_cst_time().strftime('%a, %d %b %Y %H:%M:%S GMT')

        body = ET.SubElement(opml, 'body')

        for category in categories:
            # Create category outline
            cat_outline = ET.SubElement(body, 'outline')
            cat_outline.set('text', category.name)
            cat_outline.set('title', category.name)

            # Add feeds to category
            for feed in category.feeds:
                feed_outline = ET.SubElement(cat_outline, 'outline')
                feed_outline.set('type', 'rss')
                feed_outline.set('text', feed.title)
                feed_outline.set('title', feed.title)
                feed_outline.set('xmlUrl', feed.url)
                if feed.description:
                    feed_outline.set('description', feed.description)

        # Add uncategorized feeds
        uncategorized_feeds = Feed.query.filter_by(category_id=None).all()
        if uncategorized_feeds:
            for feed in uncategorized_feeds:
                feed_outline = ET.SubElement(body, 'outline')
                feed_outline.set('type', 'rss')
                feed_outline.set('text', feed.title)
                feed_outline.set('title', feed.title)
                feed_outline.set('xmlUrl', feed.url)
                if feed.description:
                    feed_outline.set('description', feed.description)

        # Pretty print XML
        xml_str = minidom.parseString(ET.tostring(opml)).toprettyxml(indent='  ')

        return Response(
            xml_str,
            mimetype='text/xml',
            headers={
                'Content-Disposition': 'attachment; filename=feeds.opml'
            }
        )

    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500
