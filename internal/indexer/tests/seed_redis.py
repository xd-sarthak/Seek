"""
Seed script — simulates what the spider puts into Redis.

Usage:
    python tests/seed_redis.py

This pushes 3 fake crawled pages into Redis so the indexer
has something to consume.
"""
import redis
from email.utils import format_datetime
from datetime import datetime, timezone

r = redis.Redis(host="localhost", port=6379, decode_responses=True)

# ---------- PAGE 1: Normal English page ----------
page1_id = "page:test_page_1"
r.hset(page1_id, mapping={
    "normalized_url": "https://example.com/python-guide",
    "html": (
        "<html><head>"
        "<meta property='og:title' content='Python Programming Guide'>"
        "<meta name='description' content='Learn Python programming from scratch'>"
        "</head><body>"
        "<p>Python is a versatile programming language used for web development, "
        "data science, machine learning, and automation.</p>"
        "<p>Many developers choose Python because of its clean syntax and large "
        "ecosystem of libraries. Django and Flask are popular web frameworks.</p>"
        "<p>Python supports multiple paradigms including object-oriented, functional, "
        "and procedural programming styles.</p>"
        "</body></html>"
    ),
    "content_type": "text/html",
    "status_code": "200",
    "last_crawled": format_datetime(datetime.now(timezone.utc)),
})
r.sadd("outlinks:https://example.com/python-guide",
       "https://example.com/django-tutorial",
       "https://example.com/flask-tutorial",
       "https://docs.python.org")
r.lpush("pages_queue", page1_id)
print(f"✅ Seeded {page1_id}")

# ---------- PAGE 2: Another English page ----------
page2_id = "page:test_page_2"
r.hset(page2_id, mapping={
    "normalized_url": "https://example.com/machine-learning",
    "html": (
        "<html><head>"
        "<meta property='og:title' content='Machine Learning Basics'>"
        "<meta name='description' content='Introduction to ML concepts'>"
        "</head><body>"
        "<p>Machine learning is a subset of artificial intelligence that enables "
        "systems to learn from data without being explicitly programmed.</p>"
        "<p>Supervised learning, unsupervised learning, and reinforcement learning "
        "are the three main types of machine learning algorithms.</p>"
        "</body></html>"
    ),
    "content_type": "text/html",
    "status_code": "200",
    "last_crawled": format_datetime(datetime.now(timezone.utc)),
})
r.sadd("outlinks:https://example.com/machine-learning",
       "https://example.com/deep-learning",
       "https://scikit-learn.org")
r.lpush("pages_queue", page2_id)
print(f"✅ Seeded {page2_id}")

# ---------- PAGE 3: Page with NO meta tags (edge case) ----------
page3_id = "page:test_page_3"
r.hset(page3_id, mapping={
    "normalized_url": "https://example.com/minimal-page",
    "html": (
        "<html><body>"
        "<p>This is a minimal page with no meta tags at all. "
        "The indexer should still handle it gracefully without crashing.</p>"
        "</body></html>"
    ),
    "content_type": "text/html",
    "status_code": "200",
    "last_crawled": format_datetime(datetime.now(timezone.utc)),
})
# No outlinks for this page (edge case)
r.lpush("pages_queue", page3_id)
print(f"✅ Seeded {page3_id}")

# ---------- Summary ----------
queue_len = r.llen("pages_queue")
print(f"\n📊 Queue size: {queue_len}")
print("🚀 Ready to run the indexer!")
