#!/usr/bin/env python3
"""Endpoint-level call graph PDF in ACM_QUARTER style (compact paper figure).

When using ``go run ./cmd/viz -paper``, the repo root ``.venv/bin/python`` is
preferred (e.g. top-level clone like ``roshanfer-experiments/.venv``).

Reads callgraph JSON, emits unlabeled endpoint circles grouped in per-service
clusters + USER, lays out with Graphviz dot -Tjson (xdot format). Edges and
arrowheads are drawn from Graphviz's pre-computed xdot commands. Endpoints
with weighted (dynamic) fan-out get distinct fill colors; other nodes stay
neutral. No node or edge text.
"""
from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
from collections import defaultdict
from pathlib import Path

from matplotlib.lines import Line2D
from matplotlib.patches import Ellipse, PathPatch, Polygon, Rectangle
from matplotlib.path import Path as MPath

# Repo root: benchmarks/callgraph-framework/viz/ -> parents[3]
_REPO_ROOT = Path(__file__).resolve().parents[3]
if str(_REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(_REPO_ROOT))

from exec.plots.plotting_primitives import ACM_QUARTER, SubplotGrid

_CLUSTER_FILL = "#f1f3f5"


def _dot_quote(name: str) -> str:
    return '"' + name.replace("\\", "\\\\").replace('"', '\\"') + '"'


def _cluster_id(svc: str) -> str:
    safe = svc.replace("-", "_").replace(".", "_").replace(":", "_")
    return f"cluster_{safe}"


def _weighted_fanout_endpoints(data: dict) -> set[str]:
    """Endpoints with ≥2 outgoing weighted edges (dynamic fan-out)."""
    count_by_src: dict[str, int] = defaultdict(int)
    for e in data["edges"]:
        if e.get("weight") is None:
            continue
        src = e["source"]
        if src == "USER":
            continue
        count_by_src[src] += 1
    return {nid for nid, n in count_by_src.items() if n >= 2}


def _fanout_facecolors(data: dict, palette: list[str]) -> dict[str, str]:
    wf = sorted(_weighted_fanout_endpoints(data))
    return {wf[i]: palette[i % len(palette)] for i in range(len(wf))}


def _build_endpoint_graph(
    data: dict,
) -> tuple[dict[str, list[str]], dict[tuple[str, str], None]]:
    """Service -> endpoint ids, and unique endpoint-pair edges."""
    clusters: dict[str, list[str]] = {}
    for n in data["nodes"]:
        if n["id"] == "USER":
            continue
        clusters[n["id"]] = [f"{n['id']}:{iface['name']}" for iface in n.get("interfaces", [])]
    pair_seen: dict[tuple[str, str], None] = {}
    for e in data["edges"]:
        s, t = e["source"], e["target"]
        if s == t:
            continue
        pair_seen[(s, t)] = None
    return clusters, dict(pair_seen)


def _emit_dot(
    clusters: dict[str, list[str]],
    pair_info: dict[tuple[str, str], None],
    facecolors: dict[str, str],
    rankdir: str,
) -> str:
    lines = [
        "digraph G {",
        f"  rankdir={rankdir};",
        "  splines=true;",
        "  nodesep=0.28;",
        "  ranksep=0.45;",
    ]
    fc = facecolors.get("USER", "#ffffff")
    lines.append(
        f'  {_dot_quote("USER")} [shape=circle, style=filled, fillcolor="{fc}", '
        f'fixedsize=true, width=0.26, height=0.26, label=""];'
    )
    for svc in sorted(clusters):
        lines.append(f"  subgraph {_dot_quote(_cluster_id(svc))} {{")
        lines.append('    label="";')
        lines.append("    style=filled;")
        lines.append(f'    fillcolor="{_CLUSTER_FILL}";')
        lines.append('    color="#cccccc";')
        for ep in clusters[svc]:
            efc = facecolors.get(ep, "#ffffff")
            lines.append(
                f'    {_dot_quote(ep)} [shape=circle, style=filled, fillcolor="{efc}", '
                f'fixedsize=true, width=0.26, height=0.26, label=""];'
            )
        lines.append("  }")
    for (s, t) in sorted(pair_info.keys()):
        lines.append(f"  {_dot_quote(s)} -> {_dot_quote(t)};")
    lines.append("}")
    return "\n".join(lines) + "\n"


def _is_cluster(obj: dict) -> bool:
    return str(obj.get("name", "")).startswith("cluster")


def _add_legend(grid: SubplotGrid, style, lw: float) -> None:
    ms = Rectangle(
        (0, 0), 1, 1,
        facecolor=_CLUSTER_FILL, edgecolor="0.2", linewidth=lw,
    )
    ep = Line2D(
        [0], [0],
        linestyle="None",
        marker="o",
        markersize=style.marker_size,
        markerfacecolor="#ffffff",
        markeredgecolor="0.2",
        markeredgewidth=lw,
    )
    grid.add_shared_legend(
        position="top",
        handles=[ms, ep],
        labels=["Microservice", "Endpoint"],
        ncol=2,
    )


def _draw(gj: dict, facecolors: dict[str, str], out: Path) -> None:
    style = ACM_QUARTER
    grid = SubplotGrid(style, layout="1x1")
    ax = grid.get_ax(0, 0)
    ax.set_aspect("equal")
    for spine in ax.spines.values():
        spine.set_visible(False)
    ax.set_xticks([])
    ax.set_yticks([])
    lw = max(0.8, style.line_width * 0.55)

    if not gj.get("objects"):
        _add_legend(grid, style, lw)
        grid.save(out)
        return

    bb = list(map(float, gj["bb"].split(",")))  # x0, y0, x1, y1 in points
    pad = 8.0
    ax.set_xlim(bb[0] - pad, bb[2] + pad)
    ax.set_ylim(bb[1] - pad, bb[3] + pad)

    for obj in gj.get("objects", []):
        if not _is_cluster(obj):
            continue
        for cmd in obj.get("_draw_", []):
            if cmd["op"] in ("P", "p"):
                ax.add_patch(Polygon(
                    cmd["points"], closed=True,
                    facecolor=_CLUSTER_FILL, edgecolor="0.2", linewidth=lw, zorder=0,
                ))

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
        if _is_cluster(obj):
            continue
        name = obj.get("name", "")
        fill = facecolors.get(name, "#ffffff")
        for cmd in obj.get("_draw_", []):
            if cmd["op"] in ("e", "E"):
                cx, cy, rx, ry = cmd["rect"]
                ax.add_patch(Ellipse(
                    (cx, cy), width=2 * rx, height=2 * ry,
                    facecolor=fill, edgecolor="0.2", linewidth=lw, zorder=3,
                ))

    _add_legend(grid, style, lw)
    grid.save(out)


def _run_dot(dot_src: str) -> dict:
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
        return json.loads(proc.stdout)
    finally:
        Path(dot_path).unlink(missing_ok=True)


def render(callgraph_json: Path, pdf_out: Path) -> None:
    with open(callgraph_json, encoding="utf-8") as f:
        data = json.load(f)
    style = ACM_QUARTER
    fc_map = _fanout_facecolors(data, style.colors)
    clusters, pairs = _build_endpoint_graph(data)
    rankdir = "TB" if len(clusters) >= 6 else "LR"
    gj = _run_dot(_emit_dot(clusters, pairs, fc_map, rankdir))
    _draw(gj, fc_map, pdf_out)


def main() -> None:
    p = argparse.ArgumentParser(description="ACM_QUARTER endpoint-level call graph PDF")
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
