import urllib.request, json, sys
sys.stdout.reconfigure(encoding='utf-8')

r = urllib.request.urlopen('http://localhost:5000/api/topic-tags/hierarchy')
d = json.loads(r.read())

def find(nodes, tid):
    for n in nodes:
        if n['id'] == tid:
            return n
        res = find(n.get('children', []), tid)
        if res:
            return res
    return None

def search_labels(nodes, kw, path=''):
    results = []
    for n in nodes:
        p = path + ' > ' + n['label'] if path else n['label']
        if kw in n['label']:
            results.append((n['id'], n['label'], p))
        results.extend(search_labels(n.get('children', []), kw, p))
    return results

n62547 = find(d['data']['nodes'], 62547)
if n62547:
    print(f"62547 found as root: {n62547['label']}")
    print(f"  children count: {len(n62547.get('children', []))}")
    n63231 = find(n62547.get('children', []), 63231)
    if n63231:
        print(f"  63231 found as child: {n63231['label']}")
        print(f"    children count: {len(n63231.get('children', []))}")
    else:
        print("  63231 NOT found in children")
else:
    print("62547 NOT found")

print()
print("Event tags with '白宫记者':")
evts = search_labels(d['data']['nodes'], '白宫记者')
for tid, label, p in evts:
    print(f"  {tid}: {label}")
    print(f"    path: {p}")

print()
print("Event tags with '特朗普':")
evts2 = search_labels(d['data']['nodes'], '特朗普')
for tid, label, p in evts2[:10]:
    print(f"  {tid}: {label}")
    print(f"    path: {p}")
if len(evts2) > 10:
    print(f"  ... and {len(evts2)-10} more")
