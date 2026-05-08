#!/usr/bin/env python3
"""Service-level call graph PDF in ACM_QUARTER style (compact paper figure).

When using ``go run ./cmd/viz -paper``, the repo root ``.venv/bin/python`` is
preferred (e.g. top-level clone like ``roshanfer-experiments/.venv``).

Reads callgraph JSON, collapses to microservice-level nodes + USER, deduplicates
edges, lays out with Graphviz dot -Tjson (xdot format). Edges and arrowheads are drawn
directly from Graphviz's pre-computed xdot drawing commands — no manual clipping needed.
Tight nodesep/ranksep keeps the layout compact. Microservices that implement a weighted
(dynamic) fan-out get distinct fill colors; other nodes stay neutral. No edge text.
"""
from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
from collections import defaultdict
from pathlib import Path

from matplotlib.patches import Ellipse, PathPatch, Polygon
from matplotlib.path import Path as MPath

# Repo root: benchmarks/callgraph-framework/viz/ -> parents[3]
_REPO_ROOT = Path(__file__).resolve().parents[3]
if str(_REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(_REPO_ROOT))

from exec.plots.plotting_primitives import ACM_QUARTER, SubplotGrid


def _node_to_service(nid: str) -> str:
    if nid == "USER":
        return "USER"
    if ":" in nid:
        return nid.split(":", 1)[0]
    return nid


def _dot_quote(name: str) -> str:
    return '"' + name.replace("\\", "\\\\").replace('"', '\\"') + '"'


def _weighted_fanout_services(data: dict) -> set[str]:
    """Services that have at least one interface node with ≥2 outgoing weighted edges (dynamic fan-out)."""
    count_by_src_node: dict[str, int] = defaultdict(int)
    for e in data["edges"]:
        if e.get("weight") is None:
            continue
        src = e["source"]
        if src == "USER":
            continue
        count_by_src_node[src] += 1
    out: set[str] = set()
    for full_id, n in count_by_src_node.items():
        if n >= 2:
            out.add(_node_to_service(full_id))
    return out


def _fanout_facecolors(data: dict, palette: list[str]) -> dict[str, str]:
    wf = sorted(_weighted_fanout_services(data))
    return {wf[i]: palette[i % len(palette)] for i in range(len(wf))}


def _build_service_graph(data: dict) -> tuple[list[str], dict[tuple[str, str], None]]:
    """Ordered DOT node names and unique service-pair edges."""
    pair_seen: dict[tuple[str, str], None] = {}
    for e in data["edges"]:
        s = _node_to_service(e["source"])
        t = _node_to_service(e["target"])
        if s == t:
            continue
        pair_seen[(s, t)] = None
    services = {_node_to_service(n["id"]) for n in data["nodes"] if n["id"] != "USER"}
    services.add("USER")
    ordered = ["USER"] + sorted(services - {"USER"})
    return ordered, dict(pair_seen)


def _emit_dot(
    nodes: list[str], pair_info: dict[tuple[str, str], None], facecolors: dict[str, str]
) -> str:
    rankdir = "TB" if len(nodes) > 15 else "LR"
    lines = [
        "digraph G {",
        f"  rankdir={rankdir};",
        "  splines=true;",
        "  nodesep=0.28;",
        "  ranksep=0.45;",
    ]
    for name in nodes:
        fc = facecolors.get(name, "#ffffff")
        lines.append(
            f'  {_dot_quote(name)} [shape=circle, style=filled, fillcolor="{fc}", '
            f'fixedsize=true, width=0.26, height=0.26, label=""];'
        )
    for (s, t) in sorted(pair_info.keys()):
        lines.append(f"  {_dot_quote(s)} -> {_dot_quote(t)};")
    lines.append("}")
    return "\n".join(lines) + "\n"


def _draw(gj: dict, facecolors: dict[str, str], out: Path) -> None:
    style = ACM_QUARTER
    grid = SubplotGrid(style, layout="1x1")
    ax = grid.get_ax(0, 0)
    ax.set_aspect("equal")
    for spine in ax.spines.values():
        spine.set_visible(False)
    ax.set_xticks([])
    ax.set_yticks([])

    if not gj.get("objects"):
        grid.save(out)
        return

    bb = list(map(float, gj["bb"].split(",")))  # x0, y0, x1, y1 in points
    pad = 8.0
    ax.set_xlim(bb[0] - pad, bb[2] + pad)
    ax.set_ylim(bb[1] - pad, bb[3] + pad)

    lw = max(0.8, style.line_width * 0.55)

    for ed in gj.get("edges", []):
        for cmd in ed.get("_draw_", []):
            if cmd["op"] == "b":
                pts = [tuple(p) for p in cmd["points"]]
                codes = [MPath.MOVETO] + [MPath.CURVE4] * (len(pts) - 1)
                ax.add_patch(PathPatch(
                    MPath(pts, codes),
                    facecolor="none", edgecolor="0.2", linewidth=lw, zorder=1,
                ))
        for cmd in ed.get("_hdraw_", []):
            if cmd["op"] == "P":
                ax.add_patch(Polygon(
                    cmd["points"], closed=True,
                    facecolor="0.2", edgecolor="none", linewidth=0, zorder=2,
                ))

    for obj in gj.get("objects", []):
        name = obj.get("name", "")
        fill = facecolors.get(name, "#ffffff")
        for cmd in obj.get("_draw_", []):
            if cmd["op"] in ("e", "E"):
                cx, cy, rx, ry = cmd["rect"]
                ax.add_patch(Ellipse(
                    (cx, cy), width=2 * rx, height=2 * ry,
                    facecolor=fill, edgecolor="0.2", linewidth=lw, zorder=3,
                ))

    grid.save(out)


def render(callgraph_json: Path, pdf_out: Path) -> None:
    with open(callgraph_json, encoding="utf-8") as f:
        data = json.load(f)
    style = ACM_QUARTER
    fc_map = _fanout_facecolors(data, style.colors)
    nodes, pairs = _build_service_graph(data)
    dot_src = _emit_dot(nodes, pairs, fc_map)
    with tempfile.NamedTemporaryFile(mode="w", suffix=".dot", delete=False) as tf:
        tf.write(dot_src)
        dot_path = tf.name
    try:
        proc = subprocess.run(
            ["dot", "-Tjson", dot_path],
            capture_output=True,
            text=True,
            check=False,
        )
        if proc.returncode != 0:
            raise RuntimeError(
                "dot failed (install graphviz: apt install graphviz / brew install graphviz):\n"
                + (proc.stderr or proc.stdout or "")
            )
        gj = json.loads(proc.stdout)
    finally:
        Path(dot_path).unlink(missing_ok=True)
    _draw(gj, fc_map, pdf_out)


def main() -> None:
    p = argparse.ArgumentParser(description="ACM_QUARTER service-level call graph PDF")
    p.add_argument("callgraph_json", type=Path)
    p.add_argument("-o", "--output", type=Path, help="output PDF path")
    args = p.parse_args()
    out = args.output
    if out is None:
        out = args.callgraph_json.parent / "callgraph-service.pdf"
    render(args.callgraph_json, out)
    print(f"Wrote {out}", file=sys.stderr)


if __name__ == "__main__":
    main()
