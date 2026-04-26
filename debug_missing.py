"""
Check if the 7 missing roots appear anywhere in the API tree as non-root nodes.
"""
import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

with open(r'D:\project\my-robot\hierarchy_response.json', 'r', encoding='utf-8') as f:
    d = json.load(f)

missing_ids = {62039, 62049, 62365, 62633, 63231, 63651, 63899}

def find_all(nodes, target_ids, path=""):
    results = []
    for n in nodes:
        current_path = f"{path} > {n['label']}" if path else n['label']
        if n['id'] in target_ids:
            results.append((n['id'], n['label'], current_path, len(n.get('children', []))))
        results.extend(find_all(n.get('children', []), target_ids, current_path))
    return results

found = find_all(d['data']['nodes'], missing_ids)
print(f"Searching for {len(missing_ids)} missing root IDs in API tree...")
if found:
    for tid, label, path, children in found:
        print(f"  FOUND: {tid} ({label}) children={children} at: {path}")
else:
    print("  NONE of the 7 missing IDs found anywhere in the API tree")

# Also search by label fragments
print("\nSearching by label fragments:")
keywords = ['美伊冲突', '白宫记者协会晚宴', '电池技术', 'DeepSeek', '智能驾驶', '美国政治人物', '铁穹']
def search_labels(nodes, keywords, path=""):
    results = []
    for n in nodes:
        current_path = f"{path} > {n['label']}" if path else n['label']
        for kw in keywords:
            if kw in n['label']:
                results.append((n['id'], n['label'], current_path))
                break
        results.extend(search_labels(n.get('children', []), keywords, current_path))
    return results

label_matches = search_labels(d['data']['nodes'], keywords)
for tid, label, path in label_matches:
    print(f"  {tid}: {label} at: {path}")
