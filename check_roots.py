import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

with open(r'D:\project\my-robot\hierarchy_response.json', 'r', encoding='utf-8') as f:
    d = json.load(f)

print(f"success: {d['success']}")
print(f"total roots: {d['data']['total']}")
print()

print("All root nodes:")
for n in d['data']['nodes']:
    children_count = len(n.get('children', []))
    print(f"  {n['id']}: {n['label']} (children: {children_count})")
    if children_count > 0:
        for c in n['children'][:5]:
            gc_count = len(c.get('children', []))
            print(f"    {c['id']}: {c['label']} (children: {gc_count})")
        if children_count > 5:
            print(f"    ... and {children_count - 5} more")
