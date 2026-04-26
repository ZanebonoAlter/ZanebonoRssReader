"""
Simulate GetTagHierarchy cycle detection logic using direct DB queries.
"""
import subprocess
import sys
import json

def psql(sql):
    result = subprocess.run(
        ["docker", "exec", "zanebono-rssreader-pgvector", "psql", "-U", "postgres", "-d", "rss_reader", "-t", "-A", "-c", sql],
        capture_output=True, text=True, encoding="utf-8"
    )
    return result.stdout.strip()

# Step 1: Load all abstract relations
rows = psql("SELECT parent_id, child_id FROM topic_tag_relations WHERE relation_type = 'abstract'")
relations = []
for line in rows.split("\n"):
    if not line.strip():
        continue
    parts = line.split("|")
    relations.append((int(parts[0]), int(parts[1])))

print(f"Total abstract relations: {len(relations)}")

# Step 2: Build tagIDSet
tagIDSet = set()
for pid, cid in relations:
    tagIDSet.add(pid)
    tagIDSet.add(cid)

print(f"Total unique tag IDs: {len(tagIDSet)}")

# Step 3: Load active tags
tag_rows = psql(f"SELECT id, label, category, kind, source, quality_score FROM topic_tags WHERE id IN ({','.join(str(x) for x in tagIDSet)}) AND status = 'active'")
tagMap = {}
for line in tag_rows.split("\n"):
    if not line.strip():
        continue
    parts = line.split("|")
    tagMap[int(parts[0])] = {
        "id": int(parts[0]),
        "label": parts[1],
        "category": parts[2],
        "kind": parts[3],
        "source": parts[4],
        "quality_score": float(parts[5]) if parts[5] else 0
    }

print(f"Active tags in tagMap: {len(tagMap)}")

# Step 4: Check if 62547 and 63231 are in tagMap
for tid in [62547, 63231]:
    if tid in tagMap:
        print(f"  {tid}: {tagMap[tid]['label']} (category={tagMap[tid]['category']})")
    else:
        print(f"  {tid}: NOT in tagMap!")

# Step 5: Build childrenMap and parentSet (only for tags in tagMap)
childrenMap = {}
parentSet = set()
childSet = set()

for pid, cid in relations:
    child = tagMap.get(cid)
    if not child:
        continue
    childrenMap.setdefault(pid, []).append(cid)
    parentSet.add(pid)
    childSet.add(cid)

print(f"\nparentSet size: {len(parentSet)}")
print(f"childSet size: {len(childSet)}")

# Step 6: Find roots (parents NOT in childSet)
roots = [pid for pid in parentSet if pid not in childSet]
print(f"Normal roots: {len(roots)}")

# Check if 62547 and 63231 are in childSet
for tid in [62547, 63231]:
    in_parent = tid in parentSet
    in_child = tid in childSet
    print(f"  {tid}: parentSet={in_parent}, childSet={in_child}")

# Step 7: Cycle detection
if len(roots) == 0 and len(parentSet) > 0:
    print("\nNo normal roots found, running cycle detection...")

    childToParent = {}
    for pid, cid in relations:
        childToParent[cid] = pid

    # Check childToParent for key IDs
    for tid in [62547, 63231]:
        if tid in childToParent:
            print(f"  childToParent[{tid}] = {childToParent[tid]}")
        else:
            print(f"  childToParent[{tid}] = NOT FOUND")

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
    for rid in sorted(cycleRoots):
        label = tagMap.get(rid, {}).get("label", "???")
        children_count = len(childrenMap.get(rid, []))
        print(f"  {rid}: {label} ({children_count} children)")

    # Check if 62547 is a cycle root
    if 62547 in cycleRoots:
        print("\n62547 IS a cycle root")
        children = childrenMap.get(62547, [])
        print(f"  children: {len(children)}")
        for cid in children[:10]:
            clabel = tagMap.get(cid, {}).get("label", "???")
            print(f"    {cid}: {clabel}")
    else:
        print("\n62547 is NOT a cycle root")

    # Trace path from 62547
    print("\nTracing path from 62547:")
    path = set()
    current = 62547
    steps = 0
    while steps < 20:
        steps += 1
        if current in path:
            print(f"  CYCLE at {current}")
            break
        path.add(current)
        p = childToParent.get(current)
        label = tagMap.get(current, {}).get("label", "???")
        print(f"  {current}: {label} -> parent={p}")
        if p is None:
            print(f"  NO PARENT, stopping")
            break
        current = p
