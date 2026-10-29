#!/usr/bin/env python3
"""Generate the tray icon states into internal/tray/icons/.

Draws the headset-on-dock silhouette from primitives and writes black
(macOS template) and white (Linux) PNGs plus white multi-size ICOs (Windows).
Needs rsvg-convert and ImageMagick. Run: python3 icons/states/gen.py
"""
import os, subprocess, tempfile

DST = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "internal", "tray", "icons")

# --- primitives (64x64 viewBox, black on transparent) -------------------
BAND = '<path d="M11 36 V30 A21 21 0 0 1 53 30 V36" fill="none" stroke="#000" stroke-width="5.5" stroke-linecap="round"/>'
CUP_L = '<rect x="13" y="26" width="14" height="25" rx="5" fill="#000" transform="rotate(-16 20 38.5)"/>'
CUP_R = '<rect x="37" y="26" width="14" height="25" rx="5" fill="#000" transform="rotate(16 44 38.5)"/>'
DOCK = '<rect x="5" y="53" width="54" height="7" rx="2.5" fill="#000"/>'

BAND_O = '<path d="M11 36 V30 A21 21 0 0 1 53 30 V36" fill="none" stroke="#000" stroke-width="2.5" stroke-linecap="round"/>'
CUP_L_O = '<rect x="14.25" y="27.25" width="11.5" height="22.5" rx="4.5" fill="none" stroke="#000" stroke-width="2.5" transform="rotate(-16 20 38.5)"/>'
CUP_R_O = '<rect x="38.25" y="27.25" width="11.5" height="22.5" rx="4.5" fill="none" stroke="#000" stroke-width="2.5" transform="rotate(16 44 38.5)"/>'
DOCK_O = '<rect x="6.25" y="54.25" width="51.5" height="4.5" rx="2.25" fill="none" stroke="#000" stroke-width="2.5"/>'

HEADSET = BAND + CUP_L + CUP_R
HEADSET_O = BAND_O + CUP_L_O + CUP_R_O
# Dashed ghost outline for "not here".
HEADSET_DASH = HEADSET_O.replace('stroke-width="2.5"', 'stroke-width="2.5" stroke-dasharray="3.5 3"')

# Slash: knocked-out gap (mask) plus black line on top.
def slashed(body, x1=9, y1=7, x2=57, y2=61, gap=12, line=6.5):
    return f'''<defs><mask id="m"><rect width="64" height="64" fill="#fff"/>
<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="#000" stroke-width="{gap}" stroke-linecap="round"/></mask></defs>
<g mask="url(#m)">{body}</g>
<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="#000" stroke-width="{line}" stroke-linecap="round"/>'''

def dim(body, o=0.45):
    return f'<g opacity="{o}">{body}</g>'

STATES = {
    # USB receiver absent. Quietest state: dimmed ghost, no dock. Shifted
    # down so it sits centered in the viewBox without the dock.
    "nodongle": dim(f'<g transform="translate(0 4.5)">{HEADSET_DASH}</g>', 0.5),
    "connected": HEADSET + DOCK,
    "disconnected": HEADSET_O + DOCK_O,
    "muted": slashed(HEADSET + DOCK),
}

# systray fixes the macOS status image at 16pt, so the artwork must fill the
# canvas edge to edge. Crop the viewBox to the tight bounds of headset + dock.
def svg(body):
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="4.6 6.25 54.75 54.75" width="64" height="64">{body}</svg>'

with tempfile.TemporaryDirectory() as tmp:
    for state, body in STATES.items():
        src = os.path.join(tmp, f"{state}.svg")
        with open(src, "w") as f:
            f.write(svg(body))
        black = os.path.join(DST, f"{state}_black.png")
        white = os.path.join(DST, f"{state}_white.png")
        subprocess.run(["rsvg-convert", "-w", "64", "-h", "64", "-o", black, src], check=True)
        subprocess.run(["magick", black, "-channel", "RGB", "-negate", white], check=True)
        subprocess.run(["magick", white, "-define", "icon:auto-resize=64,48,32,16",
                        os.path.join(DST, f"{state}_white.ico")], check=True)
print("ok")
