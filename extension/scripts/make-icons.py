#!/usr/bin/env python3
"""Generate scry icon PNGs from a single SVG-ish recipe.

The icon is a flat coral square with a stylised eye glyph. Keeps brand
consistent across the manifest icon sizes. Run: `python3 scripts/make-icons.py`.
"""
from pathlib import Path
from PIL import Image, ImageDraw

here = Path(__file__).resolve().parent
out = here.parent / "public" / "icons"
out.mkdir(parents=True, exist_ok=True)

BG = (15, 16, 20, 255)  # matches --color-bg-base
FG = (240, 62, 47, 255)  # matches --color-accent (Sanity coral)

def make(size: int) -> Image.Image:
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    # Rounded square background
    radius = max(2, size // 6)
    draw.rounded_rectangle([(0, 0), (size - 1, size - 1)], radius=radius, fill=BG)
    # Eye: horizontal oval "cornea" + pupil
    inset_x = size * 0.18
    inset_y = size * 0.32
    cornea = [inset_x, inset_y, size - inset_x, size - inset_y]
    draw.ellipse(cornea, outline=FG, width=max(1, size // 14))
    # Pupil
    pr = size * 0.13
    cx, cy = size / 2, size / 2
    draw.ellipse(
        [cx - pr, cy - pr, cx + pr, cy + pr],
        fill=FG,
    )
    return img

for s in (16, 32, 48, 128):
    img = make(s)
    path = out / f"icon-{s}.png"
    img.save(path, "PNG")
    print(f"wrote {path}")
