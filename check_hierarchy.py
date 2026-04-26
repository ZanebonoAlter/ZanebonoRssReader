import json
import sys
sys.stdout.reconfigure(encoding='utf-8')

with open(r'D:\project\my-robot\hierarchy_response.json', 'r', encoding='utf-8') as f:
    d = json.load(f)

print(f"success: {d['success']}")
print(f"total roots: {d['data']['total']}")

def find_tags(nodes, keywords, path="", depth=0):
    for n in nodes:
        current_path = f"{path} > {n['label']}" if path else n['label']
        label = n['label']
        if any(k in label for k in keywords):
            print(f"  {'  '*depth}[{n['id']}] {label} (children: {len(n.get('children', []))}) path: {current_path}")
        children = n.get('children', [])
        if children:
            find_tags(children, keywords, current_path, depth+1)

keywords = ['枪击', '白宫记者', '晚宴', '白宫记协']
print("\nSearching for event-related tags in hierarchy:")
find_tags(d['data']['nodes'], keywords)

# Also check if 63231 and 62547 are in the tree
print("\nSearching for 63231 and 62547:")
def find_by_id(nodes, target_ids, path="", depth=0):
    for n in nodes:
        current_path = f"{path} > {n['label']}" if path else n['label']
        if n['id'] in target_ids:
            print(f"  {'  '*depth}[{n['id']}] {n['label']} (children: {len(n.get('children', []))}) path: {current_path}")
        children = n.get('children', [])
        if children:
            find_by_id(children, target_ids, current_path, depth+1)

find_by_id(d['data']['nodes'], {63231, 62547})
