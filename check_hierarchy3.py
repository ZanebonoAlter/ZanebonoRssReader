import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

with open(r'D:\project\my-robot\hierarchy_response.json', 'r', encoding='utf-8') as f:
    d = json.load(f)

# Check if 62547 or 63231 are roots (directly in nodes list)
root_ids = {n['id'] for n in d['data']['nodes']}
print(f"62547 is root: {62547 in root_ids}")
print(f"63231 is root: {63231 in root_ids}")

# Check all IDs in the entire tree
all_ids = set()
def collect_ids(nodes):
    for n in nodes:
        all_ids.add(n['id'])
        collect_ids(n.get('children', []))

collect_ids(d['data']['nodes'])
print(f"\nTotal unique IDs in tree: {len(all_ids)}")
print(f"62547 in tree: {62547 in all_ids}")
print(f"63231 in tree: {63231 in all_ids}")

# Check if the IDs are in the tree at all by looking at the raw JSON
raw = json.dumps(d)
print(f"\n62547 in raw JSON: {'62547' in raw}")
print(f"63231 in raw JSON: {'63231' in raw}")
print(f"国际军事冲突 in raw JSON: {'国际军事冲突' in raw}")
print(f"美伊冲突升级 in raw JSON: {'美伊冲突升级' in raw}")
