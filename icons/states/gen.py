#!/usr/bin/env python3
"""Generate tray icon state variants as SVG + PNG.

Draws the headset-on-dock silhouette from primitives so each state can
add, remove or knock out parts. Run: python3 icons/states/gen.py
"""
import os, subprocess

OUT = os.path.dirname(os.path.abspath(__file__))

# --- primitives (64x64 viewBox, black on transparent) -------------------
BAND = '<path d="M11 36 V30 A21 21 0 0 1 53 30 V36" fill="none" stroke="#000" stroke-width="5.5" stroke-linecap="round"/>'
CUP_L = '<rect x="13" y="26" width="14" height="25" rx="5" fill="#000" transform="rotate(-16 20 38.5)"/>'
CUP_R = '<rect x="37" y="26" width="14" height="25" rx="5" fill="#000" transform="rotate(16 44 38.5)"/>'
DOCK = '<rect x="5" y="53" width="54" height="7" rx="2.5" fill="#000"/>'

BAND_O = '<path d="M11 36 V30 A21 21 0 0 1 53 30 V36" fill="none" stroke="#000" stroke-width="2.5" stroke-linecap="round"/>'
CUP_L_O = '<rect x="14.25" y="27.25" width="11.5" height="22.5" rx="4.5" fill="none" stroke="#000" stroke-width="2.5" transform="rotate(-16 20 38.5)"/>'
CUP_R_O = '<rect x="38.25" y="27.25" width="11.5" height="22.5" rx="4.5" fill="none" stroke="#000" stroke-width="2.5" transform="rotate(16 44 38.5)"/>'
DOCK_O = '<rect x="6.25" y="54.25" width="51.5" height="4.5" rx="2.25" fill="none" stroke="#000" stroke-width="2.5"/>'

# Cradle-style dock with raised ends (from the Glif brainstorm sheet).
CRADLE = '<path d="M5 49 h5 v4 h44 v-4 h5 v8 a3 3 0 0 1 -3 3 h-48 a3 3 0 0 1 -3 -3 z" fill="#000"/>'
# Dashed ghost outline for "not here".
HEADSET = BAND + CUP_L + CUP_R
HEADSET_O = BAND_O + CUP_L_O + CUP_R_O
HEADSET_DASH = HEADSET_O.replace('stroke-width="2.5"', 'stroke-width="2.5" stroke-dasharray="3.5 3"')

# Slash: knocked-out gap (mask) plus black line on top.
def slashed(body, x1=9, y1=7, x2=57, y2=61, gap=12, line=6.5):
    return f'''<defs><mask id="m"><rect width="64" height="64" fill="#fff"/>
<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="#000" stroke-width="{gap}" stroke-linecap="round"/></mask></defs>
<g mask="url(#m)">{body}</g>
<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="#000" stroke-width="{line}" stroke-linecap="round"/>'''

# Knockout-only slash: gap with no line (quieter).
def cut(body, x1=10, y1=8, x2=56, y2=60, gap=6.5):
    return f'''<defs><mask id="m"><rect width="64" height="64" fill="#fff"/>
<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="#000" stroke-width="{gap}" stroke-linecap="round"/></mask></defs>
<g mask="url(#m)">{body}</g>'''

# Mic badge bottom-right: knockout ring, filled circle, mic glyph knocked out with slash.
def mic_badge(body, cx=49, cy=49, r=14):
    return f'''<defs><mask id="m"><rect width="64" height="64" fill="#fff"/>
<circle cx="{cx}" cy="{cy}" r="{r+3}" fill="#000"/></mask>
<mask id="mic"><rect width="64" height="64" fill="#fff"/>
<rect x="{cx-3.5}" y="{cy-9}" width="7" height="12" rx="3.5" fill="#000"/>
<path d="M{cx-7} {cy-1} a7 7 0 0 0 14 0" fill="none" stroke="#000" stroke-width="2.5"/>
<line x1="{cx}" y1="{cy+6}" x2="{cx}" y2="{cy+9.5}" stroke="#000" stroke-width="2.5"/>
<line x1="{cx-8}" y1="{cy-9}" x2="{cx+8}" y2="{cy+9}" stroke="#fff" stroke-width="5"/>
<line x1="{cx-8}" y1="{cy-9}" x2="{cx+8}" y2="{cy+9}" stroke="#000" stroke-width="2.5"/></mask></defs>
<g mask="url(#m)">{body}</g>
<circle cx="{cx}" cy="{cy}" r="{r}" fill="#000" mask="url(#mic)"/>'''

# Dot badge: small filled dot with knockout ring (status-light style).
def dot_badge(body, cx=52, cy=52, r=7):
    return f'''<defs><mask id="m"><rect width="64" height="64" fill="#fff"/>
<circle cx="{cx}" cy="{cy}" r="{r+3}" fill="#000"/></mask></defs>
<g mask="url(#m)">{body}</g>
<circle cx="{cx}" cy="{cy}" r="{r}" fill="#000"/>'''

def dim(body, o=0.45):
    return f'<g opacity="{o}">{body}</g>'

# --- icon sets ---------------------------------------------------------
# "nodongle": USB receiver absent. Quietest state: dimmed ghost, no dock.
# Shifted down so it sits centered in the viewBox without the dock.
NODONGLE = dim(f'<g transform="translate(0 4.5)">{HEADSET_DASH}</g>', 0.5)

SETS = {
    # A: fill vs outline, full diagonal slash for mute
    "A-outline": {
        "nodongle": NODONGLE,
        "connected": HEADSET + DOCK,
        "disconnected": HEADSET_O + DOCK_O,
        "muted": slashed(HEADSET + DOCK),
    },
    # B: headset leaves the dock when disconnected, mic badge for mute
    "B-dock": {
        "nodongle": NODONGLE,
        "connected": HEADSET + DOCK,
        "disconnected": DOCK + dim(HEADSET_O, 0.5),
        "muted": mic_badge(HEADSET + DOCK),
    },
    # C: same as B but empty dock only when disconnected
    "C-emptydock": {
        "nodongle": NODONGLE,
        "connected": HEADSET + DOCK,
        "disconnected": DOCK,
        "muted": slashed(HEADSET + DOCK),
    },
    # D: dimmed for disconnected, quiet knockout cut for mute
    "D-dim": {
        "nodongle": NODONGLE,
        "connected": HEADSET + DOCK,
        "disconnected": dim(HEADSET + DOCK),
        "muted": slashed(HEADSET + DOCK),
    },
    # E: no dock at all, outline when disconnected, mic badge when muted
    "E-nodock": {
        "nodongle": NODONGLE,
        "connected": HEADSET,
        "disconnected": HEADSET_O,
        "muted": mic_badge(HEADSET),
    },
    # G: cradle dock stays, dashed ghost headset when disconnected
    "G-ghost": {
        "nodongle": NODONGLE,
        "connected": HEADSET + CRADLE,
        "disconnected": HEADSET_DASH + CRADLE,
        "muted": slashed(HEADSET + CRADLE),
    },
    # H: cradle dock, ghost, mic badge
    "H-ghost-badge": {
        "nodongle": NODONGLE,
        "connected": HEADSET + CRADLE,
        "disconnected": HEADSET_DASH + CRADLE,
        "muted": slashed(HEADSET + CRADLE),
    },
    # F: dock stays, mute gets a status-dot badge
    "F-dot": {
        "nodongle": NODONGLE,
        "connected": HEADSET + DOCK,
        "disconnected": HEADSET_O + DOCK_O,
        "muted": dot_badge(HEADSET + DOCK),
    },
}

# systray fixes the macOS status image at 16pt, so the artwork must fill the
# canvas edge to edge. Crop the viewBox to the tight bounds of headset + dock.
def svg(body):
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="4.6 6.25 54.75 54.75" width="64" height="64">{body}</svg>'

for name, states in SETS.items():
    d = os.path.join(OUT, name)
    os.makedirs(d, exist_ok=True)
    for state, body in states.items():
        p = os.path.join(d, f"{state}.svg")
        with open(p, "w") as f:
            f.write(svg(body))
        subprocess.run(["rsvg-convert", "-w", "64", "-h", "64", "-o", p[:-4] + ".png", p], check=True)
print("ok")

# --- install: python3 gen.py install ---------------------------------------
# Writes the shipped sets into internal/tray/icons/<name>/ as black (macOS
# template), white (Linux) PNGs and white multi-size ICOs (Windows).
SHIPPED = [("ghost", "G-ghost"), ("outline", "A-outline"), ("dim", "D-dim")]

import sys
if len(sys.argv) > 1 and sys.argv[1] == "install":
    for name, key in SHIPPED:
        dst = os.path.join(OUT, "..", "..", "internal", "tray", "icons", name)
        os.makedirs(dst, exist_ok=True)
        for state in SETS[key]:
            src = os.path.join(OUT, key, f"{state}.svg")
            black = os.path.join(dst, f"{state}_black.png")
            white = os.path.join(dst, f"{state}_white.png")
            subprocess.run(["rsvg-convert", "-w", "64", "-h", "64", "-o", black, src], check=True)
            subprocess.run(["magick", black, "-channel", "RGB", "-negate", white], check=True)
            subprocess.run(["magick", white, "-define", "icon:auto-resize=64,48,32,16",
                            os.path.join(dst, f"{state}_white.ico")], check=True)
        print(f"installed {name}")
