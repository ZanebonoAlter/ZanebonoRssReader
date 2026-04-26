"""
Full simulation of GetTagHierarchy with cycle detection.
"""
import subprocess

def psql(sql):
    result = subprocess.run(
        ["docker", "exec", "zanebono-rssreader-pgvector", "psql", "-U", "postgres", "-d", "rss_reader", "-t", "-A", "-c", sql],
        capture_output=True, text=True, encoding="utf-8"
    )
    return result.stdout.strip()

# Load all abstract relations
rows = psql("SELECT parent_id, child_id FROM topic_tag_relations WHERE relation_type = 'abstract'")
relations = []
for line in rows.split("\n"):
    if not line.strip():
        continue
    parts = line.split("|")
    relations.append((int(parts[0]), int(parts[1])))

# Build tagIDSet
tagIDSet = set()
for pid, cid in relations:
    tagIDSet.add(pid)
    tagIDSet.add(cid)

# Load active tags
tag_rows = psql(f"SELECT id, label, category FROM topic_tags WHERE id IN ({','.join(str(x) for x in tagIDSet)}) AND status = 'active'")
tagMap = {}
for line in tag_rows.split("\n"):
    if not line.strip():
        continue
    parts = line.split("|")
    tagMap[int(parts[0])] = {"id": int(parts[0]), "label": parts[1], "category": parts[2]}

# Build childrenMap, parentSet, childSet
childrenMap = {}
parentSet = set()
childSet = set()

for pid, cid in relations:
    if cid not in tagMap:
        continue
    childrenMap.setdefault(pid, []).append(cid)
    parentSet.add(pid)
    childSet.add(cid)

# Find roots (parents NOT in childSet)
normalRoots = [pid for pid in parentSet if pid not in childSet]
print(f"Normal roots: {len(normalRoots)}")

# Cycle detection (same as Go code)
childToParent = {}
for pid, cid in relations:
    childToParent[cid] = pid

cycleRoots = set()
globalVisited = set()

for pid in sorted(parentSet):
    if pid in globalVisited:
        continue
    path = set()
    current = pid
    while True:
        if current in path:
            cycleRoots.add(current)
            break
        if current in globalVisited:
            break
        path.add(current)
        p = childToParent.get(current)
        if p is None:
            break
        current = p
    globalVisited.update(path)

print(f"Cycle roots: {len(cycleRoots)}")

# Combined roots
allRoots = set(normalRoots) | cycleRoots
print(f"Total roots (normal + cycle): {len(allRoots)}")

# Check specific IDs
for tid in [62547, 63231]:
    is_normal = tid in set(normalRoots)
    is_cycle = tid in cycleRoots
    label = tagMap.get(tid, {}).get("label", "???")
    print(f"  {tid} ({label}): normal_root={is_normal}, cycle_root={is_cycle}")

# Build full tree for cycle roots and check if event tags appear
def get_all_descendants(node_id, depth=0, visited=None):
    if visited is None:
        visited = set()
    if node_id in visited:
        return []
    visited.add(node_id)
    result = []
    for cid in childrenMap.get(node_id, []):
        if cid not in tagMap:
            continue
        label = tagMap[cid]["label"]
        result.append((depth, cid, label))
        result.extend(get_all_descendants(cid, depth + 1, visited))
    return result

print("\n--- Children of 62547 (cycle root) ---")
descendants = get_all_descendants(62547)
print(f"Total descendants: {len(descendants)}")
for depth, cid, label in descendants[:20]:
    print(f"  {'  '*depth}{cid}: {label}")
if len(descendants) > 20:
    print(f"  ... and {len(descendants) - 20} more")

# Check if event tags are in the tree
print("\n--- Checking event tag presence ---")
event_keywords = ['枪击', '白宫记者协会晚宴枪击', '白宫记协']
for depth, cid, label in descendants:
    for kw in event_keywords:
        if kw in label:
            print(f"  FOUND at depth {depth}: {cid}: {label}")
            break

# Also check: what if we filter by category=event?
print("\n--- Filtering by category=event ---")
# In the Go code, category filter happens AFTER loading relations and tags
# It filters relations where parent.Category == category
event_relations = []
for pid, cid in relations:
    parent = tagMap.get(pid)
    if parent and parent["category"] == "event":
        event_relations.append((pid, cid))

event_parentSet = set()
event_childSet = set()
event_childrenMap = {}
for pid, cid in event_relations:
    if cid not in tagMap:
        continue
    event_childrenMap.setdefault(pid, []).append(cid)
    event_parentSet.add(pid)
    event_childSet.add(cid)

event_normalRoots = [pid for pid in event_parentSet if pid not in event_childSet]
print(f"Event normal roots: {len(event_normalRoots)}")

# Cycle detection for event category
event_childToParent = {}
for pid, cid in event_relations:
    event_childToParent[cid] = pid

event_cycleRoots = set()
event_globalVisited = set()

for pid in sorted(event_parentSet):
    if pid in event_globalVisited:
        continue
    path = set()
    current = pid
    while True:
        if current in path:
            event_cycleRoots.add(current)
            break
        if current in event_globalVisited:
            break
        path.add(current)
        p = event_childToParent.get(current)
        if p is None:
            break
        current = p
    event_globalVisited.update(path)

print(f"Event cycle roots: {len(event_cycleRoots)}")
event_allRoots = set(event_normalRoots) | event_cycleRoots
print(f"Event total roots: {len(event_allRoots)}")

for tid in [62547, 63231]:
    is_normal = tid in set(event_normalRoots)
    is_cycle = tid in event_cycleRoots
    label = tagMap.get(tid, {}).get("label", "???")
    print(f"  {tid} ({label}): normal_root={is_normal}, cycle_root={is_cycle}")
