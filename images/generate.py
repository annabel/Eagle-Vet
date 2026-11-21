"""Generate or verify the production image assets from local masters."""

import argparse
from io import BytesIO
import json
import math
import os
from pathlib import Path
import re
import stat
import sys
from tempfile import NamedTemporaryFile
import warnings
from xml.etree import ElementTree

from PIL import Image, ImageCms, ImageOps

CATALOG = Path("images/catalog.json")
SOURCES = Path("images/sources")
OUTPUT = Path("static/assets/images")
MANIFEST = "manifest.json"
RASTER_PRESETS = {
    "hero": (400, 700, 1000, 1400),
    "inline": (320, 560, 900, 1200),
    "thumbnail": (180, 360, 540),
    "portrait": (360, 620, 900),
    "detail": (100, 200, 320, 480),
    "brand": (128, 256, 400, 500),
    "icon": (48,),
}
SVG_NUMBER = r"[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?"
SVG_LENGTH = re.compile(rf"({SVG_NUMBER})(?:px)?")
SVG_SEPARATOR = r"(?:\s*,\s*|\s+)"
SVG_VIEW_BOX = re.compile(SVG_SEPARATOR.join([rf"({SVG_NUMBER})"] * 4))


def require_directory(path):
    if not stat.S_ISDIR(path.lstat().st_mode):
        raise ValueError(f"{path}: expected a directory, not a symlink or other file")


def require_file(path):
    if not stat.S_ISREG(path.lstat().st_mode):
        raise ValueError(f"{path}: expected a regular file, not a symlink or other entry")


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"{CATALOG}: duplicate key {key!r}")
        result[key] = value
    return result


def catalog_entries():
    require_directory(CATALOG.parent)
    require_directory(SOURCES)
    require_file(CATALOG)
    catalog = json.loads(CATALOG.read_text(encoding="utf-8"), object_pairs_hook=unique_object)
    if not isinstance(catalog, dict):
        raise ValueError(f"{CATALOG}: expected an object keyed by image ID")
    for image_id, entry in sorted(catalog.items()):
        if not re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", image_id):
            raise ValueError(f"{CATALOG}: invalid image ID {image_id!r}")
        if not isinstance(entry, dict) or entry.keys() - {"source", "provenance", "presets"}:
            raise ValueError(f"{image_id}: expected a catalog entry with source and presets")
        filename = entry.get("source")
        if (not isinstance(filename, str) or filename in {"", ".", ".."}
                or any(c in filename for c in ("/", "\\", "\x00"))):
            raise ValueError(f"{image_id}: source must be a local filename")
        if "provenance" in entry and (not isinstance(entry["provenance"], str)
                                      or not entry["provenance"].strip()):
            raise ValueError(f"{image_id}: provenance must be nonempty text when provided")
        presets = entry.get("presets")
        if not isinstance(presets, list) or not all(isinstance(p, str) for p in presets):
            raise ValueError(f"{image_id}: presets must be an array of strings")
        for preset in presets:
            if preset != "vector" and preset not in RASTER_PRESETS:
                raise ValueError(f"{image_id}: unknown preset {preset!r}")
        if len(set(presets)) != len(presets):
            raise ValueError(f"{image_id}: duplicate preset")
        if "vector" in presets and len(presets) != 1:
            raise ValueError(f"{image_id}: the vector preset cannot be combined with raster presets")
        source = SOURCES / filename
        require_file(source)
        yield image_id, source, presets


def vector_size(data):
    root = ElementTree.fromstring(data)
    if root.tag != "{http://www.w3.org/2000/svg}svg":
        raise ValueError("expected an SVG document")
    view_box = None
    if "viewBox" in root.attrib:
        match = SVG_VIEW_BOX.fullmatch(root.attrib["viewBox"].strip())
        if not match:
            raise ValueError("invalid SVG viewBox")
        view_box = tuple(map(float, match.groups()))
        if not all(map(math.isfinite, view_box)) or min(view_box[2:]) <= 0:
            raise ValueError("invalid SVG viewBox")
    width, height = (SVG_LENGTH.fullmatch(root.get(key, "").strip()) for key in ("width", "height"))
    if width and height:
        size = (float(width[1]), float(height[1]))
    elif view_box:
        size = view_box[2:]
    else:
        raise ValueError("SVG needs a bounded viewBox or explicit pixel dimensions")
    if not all(0 < side <= 2**31 - 1 for side in size):
        raise ValueError("SVG dimensions are out of range")
    return tuple(max(1, round(side)) for side in size)


def raster_pixels(path):
    with path.open("rb") as file:
        # An extended WebP can be flagged as animated even with only one frame.
        header = file.read(21)
        if (len(header) == 21 and header[:4] == b"RIFF"
                and header[8:16] == b"WEBPVP8X" and header[20] & 0x02):
            raise ValueError("animated images are not supported")

        # Pillow selects from the permitted formats by contents, not the filename.
        with Image.open(file, formats=("JPEG", "PNG", "WEBP")) as source:
            if getattr(source, "is_animated", False) or source.get_format_mimetype() == "image/apng":
                raise ValueError("animated images are not supported")
            ImageOps.exif_transpose(source, in_place=True)
            return normalize_pixels(source)


def normalize_pixels(source):
    icc_data = source.info.get("icc_profile")
    profile = ImageCms.ImageCmsProfile(BytesIO(icc_data)) if icc_data else None

    alpha = None
    if "A" in source.getbands():
        alpha = source.getchannel("A")
    elif "transparency" in source.info:
        alpha = source.convert("RGBA").getchannel("A")

    if source.mode == "I;16":
        # Map 16-bit grayscale to eight bits instead of clipping values above 255.
        color = source.point(lambda value: value / 257).convert("L")
    elif source.mode in ("CMYK", "RGB", "L"):
        color = source
    elif source.mode in ("1", "LA") or (profile and profile.profile.xcolor_space.strip() == "GRAY"):
        color = source.convert("L")
    else:
        color = source.convert("RGB")
    if profile:
        color = ImageCms.profileToProfile(color, profile, ImageCms.createProfile("sRGB"), outputMode="RGB")
    elif color.mode != "RGB":
        color = color.convert("RGB")
    if alpha is not None:
        color.putalpha(alpha)
    color.info.clear()
    return color


def raster_variants(image_id, path, presets, files):
    image = raster_pixels(path)
    variants = {}
    for preset in presets:
        ext = "png" if preset == "icon" else "webp"
        candidates = []
        for width in sorted({min(size, image.width) for size in RASTER_PRESETS[preset]}):
            height = max(1, (image.height * width + image.width // 2) // image.width)
            filename = f"{image_id}-{width}.{ext}"
            if filename not in files:
                pixels = image if width == image.width else image.resize((width, height), Image.Resampling.LANCZOS)
                with BytesIO() as buffer:
                    if ext == "png":
                        pixels.save(buffer, "PNG", compress_level=9)
                    else:
                        pixels.save(buffer, "WEBP", quality=82, method=4)
                    files[filename] = buffer.getvalue()
            candidates.append({"file": filename, "width": width, "height": height})
        variants[preset] = candidates
    return variants


def stage_assets():
    files = {}
    manifest = {}
    for image_id, source, presets in catalog_entries():
        if not presets:
            continue
        try:
            if presets == ["vector"]:
                data = source.read_bytes()
                width, height = vector_size(data)
                filename = f"{image_id}.svg"
                files[filename] = data
                manifest[image_id] = {"vector": [{"file": filename, "width": width, "height": height}]}
            else:
                manifest[image_id] = raster_variants(image_id, source, presets, files)
        except (OSError, EOFError, SyntaxError, ValueError, ImageCms.PyCMSError,
                Image.DecompressionBombError, Image.DecompressionBombWarning) as error:
            raise ValueError(f"{image_id} ({source}): {error}") from error
    files[MANIFEST] = (json.dumps(manifest, sort_keys=True, indent=2) + "\n").encode("utf-8")
    return files


def existing_assets():
    # Reject symlinks in the entire output path before reading, writing or unlinking anything.
    try:
        for directory in (*reversed(OUTPUT.parents), OUTPUT):
            require_directory(directory)
    except FileNotFoundError:
        return set()
    names = set()
    for path in sorted(OUTPUT.iterdir()):
        require_file(path)
        names.add(path.name)
    return names


def replace_file(path, data):
    with NamedTemporaryFile(dir=OUTPUT, prefix=".image-", delete=False) as temporary:
        name = Path(temporary.name)
        try:
            temporary.write(data)
            os.fchmod(temporary.fileno(), 0o644)
            temporary.close()
            name.replace(path)
        finally:
            name.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("generate", "check"))
    args = parser.parse_args()
    warnings.simplefilter("error", Image.DecompressionBombWarning)
    try:
        expected = stage_assets()
        actual = existing_assets()
        missing = sorted(expected.keys() - actual)
        obsolete = sorted(actual - expected.keys())
        different = sorted(name for name in actual & expected.keys()
                           if (OUTPUT / name).read_bytes() != expected[name])
        if args.command == "check":
            if missing or different or obsolete:
                for label, names in (("missing", missing), ("changed", different), ("obsolete", obsolete)):
                    for name in names:
                        print(f"{label}: {OUTPUT / name}", file=sys.stderr)
                print("Run ./scripts/images generate to refresh the image assets.", file=sys.stderr)
                return 1
        elif missing or different or obsolete:
            OUTPUT.mkdir(parents=True, exist_ok=True)
            # Publish the manifest only after all newly referenced files are in place.
            for name in sorted(missing + different, key=lambda name: (name == MANIFEST, name)):
                replace_file(OUTPUT / name, expected[name])
            for name in obsolete:
                (OUTPUT / name).unlink()
    except (OSError, ValueError) as error:
        print(f"images: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
