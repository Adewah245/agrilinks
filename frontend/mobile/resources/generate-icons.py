"""Regenerate native branding from the existing Agrilinks shield SVGs.

Run with Python, Pillow, and ImageMagick installed. Native builds use the
committed PNGs and do not need these image tools.
"""

from pathlib import Path
import re
import subprocess
import tempfile
import xml.etree.ElementTree as ET

from PIL import Image, ImageDraw


frontend = Path(__file__).resolve().parents[2]
resources = Path(__file__).resolve().parent
android = frontend / "android/app/src/main/res"
ios = frontend / "ios/App/App/Assets.xcassets"

with tempfile.TemporaryDirectory(prefix="agrilinks-icons-") as temporary:
    outputs = {}
    for name in ("icon", "icon-foreground"):
        target = Path(temporary) / f"{name}.png"
        subprocess.run(["convert", "-background", "none", str(resources / f"{name}.svg"), str(target)], check=True)
        outputs[name] = Image.open(target).convert("RGBA")

    for target in android.rglob("ic_launcher*.png"):
        with Image.open(target) as original:
            size = original.size
        image = outputs["icon-foreground" if "foreground" in target.name else "icon"].copy()
        if "round" in target.name:
            mask = Image.new("L", image.size)
            ImageDraw.Draw(mask).ellipse((0, 0, image.width - 1, image.height - 1), fill=255)
            image.putalpha(mask)
        image.resize(size, Image.Resampling.LANCZOS).save(target, optimize=True)

    outputs["icon"].convert("RGB").save(ios / "AppIcon.appiconset/AppIcon-512@2x.png", optimize=True)

    for target in list(android.rglob("splash.png")) + list((ios / "Splash.imageset").glob("*.png")):
        with Image.open(target) as original:
            size = original.size
        canvas = Image.new("RGB", size, "#f7f9f4")
        side = max(64, min(size) // 5)
        mark = outputs["icon"].resize((side, side), Image.Resampling.LANCZOS)
        mask = Image.new("L", (side, side))
        ImageDraw.Draw(mask).rounded_rectangle((0, 0, side - 1, side - 1), radius=side // 5, fill=255)
        canvas.paste(mark, ((size[0] - side) // 2, (size[1] - side) // 2), mask)
        canvas.save(target, optimize=True)

android_namespace = "http://schemas.android.com/apk/res/android"
ET.register_namespace("android", android_namespace)
attribute = lambda key: f"{{{android_namespace}}}{key}"
svg_namespace = "{http://www.w3.org/2000/svg}"
svg = ET.parse(resources / "icon-foreground.svg").getroot()
group = svg.find(f"{svg_namespace}g")
transform = re.fullmatch(r"translate\(([\d.]+)[ ,]+([\d.]+)\) scale\(([\d.]+)\)", group.attrib["transform"])
scale = 24 / float(svg.attrib["width"])
vector = ET.Element("vector", {attribute("width"): "108dp", attribute("height"): "108dp", attribute("viewportWidth"): "24", attribute("viewportHeight"): "24"})
native_group = ET.SubElement(vector, "group", {attribute("translateX"): str(float(transform[1]) * scale), attribute("translateY"): str(float(transform[2]) * scale), attribute("scaleX"): str(float(transform[3]) * scale), attribute("scaleY"): str(float(transform[3]) * scale)})
for path in group.findall(f"{svg_namespace}path"):
    fill = path.attrib.get("fill", group.attrib.get("fill", "none"))
    ET.SubElement(native_group, "path", {attribute("pathData"): path.attrib["d"], attribute("fillColor"): "#00000000" if fill == "none" else fill, attribute("strokeColor"): path.attrib.get("stroke", group.attrib["stroke"]), attribute("strokeWidth"): group.attrib["stroke-width"], attribute("strokeLineCap"): "round", attribute("strokeLineJoin"): "round"})
ET.indent(vector)
ET.ElementTree(vector).write(android / "drawable-v24/ic_launcher_foreground.xml", encoding="utf-8", xml_declaration=True)
(android / "drawable/ic_launcher_background.xml").write_text('<?xml version="1.0" encoding="utf-8"?>\n<shape xmlns:android="http://schemas.android.com/apk/res/android" android:shape="rectangle">\n    <solid android:color="#1b5e20" />\n</shape>\n')
