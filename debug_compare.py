"""
Compare API response with simulation to find missing roots.
"""
import subprocess
import json
import sys

def psql(sql):
    result = subprocess.run(
        ["docker", "exec", "zanebono-rssreader-pgvector", "psql", "-U", "postgres", "-d", "rss_reader", "-t", "-A", "-c", sql],
        capture_output=True, text=True, encoding="utf-8"
    )
    return result.stdout.strip()

sys.stdout.reconfigure(encoding='utf-8')

# Load API response
with open(r'D:\project\my-robot\hierarchy_response.json', 'r', encoding='utf-8') as f:
    api_data = json.load(f)

api_root_ids = set()
def collect_api_ids(nodes):
    for n in nodes:
        api_root_ids.add(n['id'])
        collect_api_ids(n.get('children', []))

collect_api_ids(api_data['data']['nodes'])
api_root_level_ids = {n['id'] for n in api_data['data']['nodes']}

print(f"API root-level nodes: {len(api_root_level_ids)}")
print(f"API total nodes in tree: {len(api_root_ids)}")

# Load all abstract relations
rows = psql("SELECT parent_id, child_id FROM topic_tag_relations WHERE relation_type = 'abstract'")
relations = []
for line in rows.split("\n"):
    if not line.strip():
        continue
    parts = line.split("|")
    relations.append((int(parts[0]), int(parts[1])))

# Build tagIDSet and load active tags
tagIDSet = set()
for pid, cid in relations:
    tagIDSet.add(pid)
    tagIDSet.add(cid)

tag_rows = psql(f"SELECT id, label, category FROM topic_tags WHERE id IN ({','.join(str(x) for x in tagIDSet)}) AND status = 'active'")
tagMap = {}
for line in tag_rows.split("\n"):
    if not line.strip():
        continue
    parts = line.split("|")
    tagMap[int(parts[0])] = {"label": parts[1], "category": parts[2]}

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

# Normal roots
normalRoots = set(pid for pid in parentSet if pid not in childSet)

# Cycle detection
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

sim_root_ids = normalRoots | cycleRoots

print(f"\nSimulation roots: {len(sim_root_ids)}")
print(f"API root-level: {len(api_root_level_ids)}")

# Find differences
in_sim_not_api = sim_root_ids - api_root_level_ids
in_api_not_sim = api_root_level_ids - sim_root_ids

print(f"\nIn simulation but NOT in API ({len(in_sim_not_api)}):")
for tid in sorted(in_sim_not_api):
    label = tagMap.get(tid, {}).get("label", "???")
    cat = tagMap.get(tid, {}).get("category", "???")
    is_cycle = tid in cycleRoots
    print(f"  {tid}: {label} (category={cat}, cycle_root={is_cycle})")

print(f"\nIn API but NOT in simulation ({len(in_api_not_sim)}):")
for tid in sorted(in_api_not_sim):
    label = tagMap.get(tid, {}).get("label", "???")
    cat = tagMap.get(tid, {}).get("category", "???")
    print(f"  {tid}: {label} (category={cat})")

# Check: does 62547 appear ANYWHERE in the API tree?
print(f"\n62547 in API tree: {62547 in api_root_ids}")
print(f"63231 in API tree: {63231 in api_root_ids}")

# Search for 62547 children in API tree
def find_node(nodes, target_id):
    for n in nodes:
        if n['id'] == target_id:
            return n
        result = find_node(n.get('children', []), target_id)
        if result:
            return result
    return None

node_63231 = find_node(api_data['data']['nodes'], 63231)
if node_63231:
    print(f"\n63231 found in API tree: {node_63231['label']}")
    print(f"  children: {len(node_63231.get('children', []))}")
else:
    print("\n63231 NOT found in API tree")

node_62547 = find_node(api_data['data']['nodes'], 62547)
if node_62547:
    print(f"\n62547 found in API tree: {node_62547['label']}")
else:
    print("\n62547 NOT found in API tree")
