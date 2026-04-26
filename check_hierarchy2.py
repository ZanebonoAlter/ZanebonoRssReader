import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

with open(r'D:\project\my-robot\hierarchy_response.json', 'r', encoding='utf-8') as f:
    d = json.load(f)

def search_tree(nodes, target_id, path=""):
    for n in nodes:
        current_path = f"{path} > {n['label']}" if path else n['label']
        if n['id'] == target_id:
            print(f"FOUND [{n['id']}] {n['label']} at path: {current_path}")
            return True
        if search_tree(n.get('children', []), target_id, current_path):
            return True
    return False

def get_tree_depth(nodes, depth=0):
    max_d = depth
    for n in nodes:
        d = get_tree_depth(n.get('children', []), depth + 1)
        max_d = max(max_d, d)
    return max_d

print(f"Total roots: {d['data']['total']}")
print(f"Max tree depth: {get_tree_depth(d['data']['nodes'])}")

# Search for specific IDs
for target_id in [62547, 63231, 63166, 63440, 63194, 62743]:
    found = search_tree(d['data']['nodes'], target_id)
    if not found:
        print(f"NOT FOUND [{target_id}]")
