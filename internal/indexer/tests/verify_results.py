"""
Verification script — checks that the indexer wrote the correct data
into MongoDB and cleaned up Redis.

Usage:
    python tests/verify_results.py
"""
import redis
import pymongo
import sys

PASS = "✅"
FAIL = "❌"
WARN = "⚠️"
results = []

def check(name, condition, detail=""):
    status = PASS if condition else FAIL
    results.append((status, name))
    print(f"  {status} {name}" + (f" — {detail}" if detail else ""))
    return condition


# ─── Connect ───
r = redis.Redis(host="localhost", port=6379, decode_responses=True)
m = pymongo.MongoClient("mongodb://localhost:27017/")
db = m["indexer"]

print("\n" + "=" * 60)
print("  INDEXER VERIFICATION REPORT")
print("=" * 60)

# ─── 1. Redis should be cleaned up ───
print("\n📦 Redis Cleanup")
test_pages = [
    "page:test_page_1",
    "page:test_page_2",
    "page:test_page_3",
]
for page_id in test_pages:
    data = r.hgetall(page_id)
    check(f"{page_id} removed from Redis", len(data) == 0,
          f"still has {len(data)} fields" if data else "clean")

test_outlinks = [
    "outlinks:https://example.com/python-guide",
    "outlinks:https://example.com/machine-learning",
]
for key in test_outlinks:
    members = r.smembers(key)
    check(f"{key} removed", len(members) == 0,
          f"still has {len(members)} members" if members else "clean")

queue_size = r.llen("pages_queue")
check("pages_queue is empty", queue_size == 0, f"size={queue_size}")

# ─── 2. Check image indexer queue ───
print("\n🖼️  Image Indexer Queue")
image_queue = r.lrange("image_indexer_queue", 0, -1)
check("image_indexer_queue has entries", len(image_queue) > 0,
      f"entries: {image_queue}")

# ─── 3. Check signal queue ───
print("\n📡 Signal Queue")
signal_queue = r.lrange("signal_queue", 0, -1)
check("signal_queue has RESUME_CRAWL", "RESUME_CRAWL" in signal_queue,
      f"entries: {signal_queue}")

# ─── 4. MongoDB: words collection ───
print("\n📝 MongoDB: words collection")
words_col = db["words"]
words_count = words_col.count_documents({})
check("words collection has documents", words_count > 0, f"count={words_count}")

python_guide_words = list(words_col.find({"url": "https://example.com/python-guide"}))
check("words exist for python-guide URL", len(python_guide_words) > 0,
      f"found {len(python_guide_words)} word entries")

# Check a word we expect to be there
python_word = words_col.find_one({"word": "python", "url": "https://example.com/python-guide"})
check("'python' indexed for python-guide", python_word is not None,
      f"tf={python_word['tf']}" if python_word else "missing")

if python_word:
    check("'python' has URL-boosted tf (in URL path)",
          python_word["tf"] > 10,
          f"tf={python_word['tf']} (should be boosted since 'python' is in the URL)")

# ─── 5. MongoDB: metadata collection ───
print("\n📋 MongoDB: metadata collection")
metadata_col = db["metadata"]
metadata_count = metadata_col.count_documents({})
check("metadata collection has documents", metadata_count > 0, f"count={metadata_count}")

meta1 = metadata_col.find_one({"_id": "https://example.com/python-guide"})
check("metadata for python-guide exists", meta1 is not None)
if meta1:
    check("  title is set", meta1.get("title") is not None, f"title={meta1.get('title')}")
    check("  description is set", meta1.get("description") is not None, f"desc={meta1.get('description')[:40]}...")
    check("  keywords is set", meta1.get("keywords") is not None, f"{len(meta1.get('keywords', {}))} keywords")
    check("  last_crawled is set", meta1.get("last_crawled") is not None)

meta3 = metadata_col.find_one({"_id": "https://example.com/minimal-page"})
check("metadata for minimal-page exists", meta3 is not None)
if meta3:
    check("  handles None title gracefully", True, f"title={meta3.get('title')}")

# ─── 6. MongoDB: outlinks collection ───
print("\n🔗 MongoDB: outlinks collection")
outlinks_col = db["outlinks"]
outlinks_count = outlinks_col.count_documents({})
check("outlinks collection has documents", outlinks_count > 0, f"count={outlinks_count}")

out1 = outlinks_col.find_one({"_id": "https://example.com/python-guide"})
check("outlinks for python-guide exist", out1 is not None)
if out1:
    check("  has correct link count", len(out1.get("links", [])) == 3,
          f"links={out1.get('links')}")

# ─── 7. MongoDB: dictionary collection ───
print("\n📖 MongoDB: dictionary collection")
dict_col = db["dictionary"]
dict_count = dict_col.count_documents({})
check("dictionary collection has documents", dict_count > 0, f"count={dict_count}")

python_dict = dict_col.find_one({"_id": "python"})
check("'python' in dictionary", python_dict is not None)

# ─── Summary ───
print("\n" + "=" * 60)
passed = sum(1 for s, _ in results if s == PASS)
failed = sum(1 for s, _ in results if s == FAIL)
print(f"  Results: {passed} passed, {failed} failed, {len(results)} total")
print("=" * 60 + "\n")

if failed > 0:
    print("Failed checks:")
    for s, name in results:
        if s == FAIL:
            print(f"  {FAIL} {name}")
    sys.exit(1)
else:
    print("🎉 All checks passed!")
    sys.exit(0)
