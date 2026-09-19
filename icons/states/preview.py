#!/usr/bin/env python3
"""Build icons/states/preview.html: all sets at menubar scale, light + dark."""
import os, re, base64
from gen import SETS, svg

OUT = os.path.dirname(os.path.abspath(__file__))

def data(body, color=None):
    return "data:image/svg+xml;base64," + base64.b64encode(svg(body).encode()).decode()

current = base64.b64encode(open(os.path.join(OUT, "..", "..", "icons/icon_black.png"), "rb").read()).decode()
current_w = base64.b64encode(open(os.path.join(OUT, "..", "..", "icons/icon_white.png"), "rb").read()).decode()

def bar(theme, cells):
    bg, fg = ("#e8e8ea", "#000") if theme == "light" else ("#2a2a2c", "#fff")
    inv = ' class="inv"' if theme == "dark" else ""
    icons = "".join(f'<img src="{c}" width="22" height="22" title="{t}"{inv}>' for c, t in cells)
    return f'''<div class="bar" style="background:{bg};color:{fg}">
      {icons}<span class="sys">􀙇</span><span class="sys">􀛨 84%</span><span class="sys">Thu 9:41 AM</span></div>'''

rows = []
for name, states in SETS.items():
    big = "".join(f'<figure><img src="{data(b, "#000")}" width="64" height="64"><figcaption>{s}</figcaption></figure>' for s, b in states.items())
    light = bar("light", [(data(b, "#000"), s) for s, b in states.items()])
    dark = bar("dark", [(data(b), s) for s, b in states.items()])
    rows.append(f'''<section><h2>{name}</h2><div class="big">{big}</div>{light}{dark}</section>''')

cur = f'''<section><h2>current</h2><div class="big"><figure><img src="data:image/png;base64,{current}" width="64" height="64"><figcaption>icon_black.png</figcaption></figure></div>
{bar("light", [("data:image/png;base64," + current, "current")])}{bar("dark", [("data:image/png;base64," + current, "current")])}</section>'''

zoom = "".join(f'<h3>{n}</h3>{bar("light", [(data(b), s) for s, b in SETS[n].items()])}{bar("dark", [(data(b), s) for s, b in SETS[n].items()])}' for n in ["H-ghost-badge","A-outline","G-ghost","D-dim"])
glif = base64.b64encode(open(os.path.join(OUT, "glif_brainstorm_v2.png"), "rb").read()).decode()
html = f'''<!doctype html><meta charset="utf-8"><title>Tray icon states</title>
<style>
body{{font:14px -apple-system,system-ui,sans-serif;background:#fafafa;color:#222;margin:24px;max-width:900px}}
section{{margin-bottom:36px}} h3{{font-size:8px;margin:6px 0 2px;font-family:ui-monospace,monospace}} h2{{font-size:15px;margin:0 0 8px;font-family:ui-monospace,monospace}}
.big{{display:flex;gap:24px;margin-bottom:10px}} figure{{margin:0;text-align:center}} figcaption{{font-size:11px;color:#777}}
.bar{{display:flex;align-items:center;gap:14px;padding:0 12px;height:24px;border-radius:6px;margin-bottom:6px;font-size:13px}}
.bar img{{display:block}} .bar img.inv{{filter:invert(1)}} .sys{{opacity:.85}}
</style>
<h1 style="font-size:18px">scapectl tray icon states</h1>
<p>Each row: 64px source, then the three states at 22px on a light and dark menubar (macOS template rendering simulated).</p>
{cur}{"".join(rows)}
<section><h2>2x zoom (top picks)</h2><div style="zoom:2;max-width:440px">{zoom}</div></section>
<section><h2>Glif brainstorm (reference)</h2><img src="data:image/png;base64,{glif}" width="600"></section>'''
open(os.path.join(OUT, "preview.html"), "w").write(html)
print("wrote preview.html")
