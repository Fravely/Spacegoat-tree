"""Genera diagramas PNG para el informe tecnico."""
from pathlib import Path
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch

OUT = Path(__file__).resolve().parent.parent / "diagrams"
OUT.mkdir(parents=True, exist_ok=True)


def architecture():
    fig, ax = plt.subplots(figsize=(10, 5))
    ax.set_xlim(0, 10)
    ax.set_ylim(0, 5)
    ax.axis("off")
    layers = [
        (4.2, "web/ — Vue.js 3 (simulacion visual)", "#4b6cb7"),
        (3.3, "cmd/simulation — API REST Go (:8080)", "#3157d5"),
        (2.4, "database/ — SQL Server (go-mssqldb)", "#7a5af8"),
        (1.5, "scapegoat/ — Scapegoat Tree en memoria", "#12b76a"),
    ]
    for y, label, color in layers:
        box = FancyBboxPatch((0.5, y - 0.35), 9, 0.7, boxstyle="round,pad=0.05", fc=color, ec="white", alpha=0.9)
        ax.add_patch(box)
        ax.text(5, y, label, ha="center", va="center", color="white", fontsize=11, fontweight="bold")
    for y in [3.75, 2.85, 1.95]:
        ax.annotate("", xy=(5, y - 0.35), xytext=(5, y - 0.65), arrowprops=dict(arrowstyle="->", color="#475467", lw=2))
    ax.set_title("Arquitectura en capas del sistema", fontsize=14, fontweight="bold", pad=12)
    fig.savefig(OUT / "01-arquitectura.png", dpi=180, bbox_inches="tight", facecolor="white")
    plt.close(fig)


def insertion_flow():
    fig, ax = plt.subplots(figsize=(11, 7))
    ax.set_xlim(0, 11)
    ax.set_ylim(0, 8)
    ax.axis("off")
    steps = [
        (5.5, 7.2, "Insercion BST estandar"),
        (5.5, 6.0, "depth > floor(log_{1/a}(q))?"),
        (2.0, 4.6, "No: fin"),
        (9.0, 4.6, "Si: ascender camino"),
        (9.0, 3.2, "size(hijo) > a*size(padre)?"),
        (9.0, 1.8, "Scapegoat = padre"),
        (9.0, 0.6, "flatten + buildBalanced"),
    ]
    for x, y, text in steps:
        fc = "#fffaeb" if "?" in text else "#eff4ff"
        if "Scapegoat" in text or "flatten" in text:
            fc = "#fff1f3"
        box = FancyBboxPatch((x - 2.2, y - 0.35), 4.4, 0.7, boxstyle="round,pad=0.04", fc=fc, ec="#344054")
        ax.add_patch(box)
        ax.text(x, y, text, ha="center", va="center", fontsize=9)
    arrows = [(5.5, 6.65, 5.5, 6.35), (5.5, 5.65, 2.0, 5.0), (5.5, 5.65, 9.0, 5.0),
              (9.0, 4.25, 9.0, 3.55), (9.0, 2.85, 9.0, 2.15), (9.0, 1.45, 9.0, 0.95)]
    for x1, y1, x2, y2 in arrows:
        ax.annotate("", xy=(x2, y2), xytext=(x1, y1), arrowprops=dict(arrowstyle="->", color="#475467"))
    ax.text(3.2, 5.3, "No", fontsize=8, color="#027a48")
    ax.text(7.0, 5.3, "Si", fontsize=8, color="#b42318")
    ax.set_title("Flujo de insercion con deteccion de Scapegoat", fontsize=14, fontweight="bold")
    fig.savefig(OUT / "02-flujo-insercion.png", dpi=180, bbox_inches="tight", facecolor="white")
    plt.close(fig)


def data_flow():
    fig, ax = plt.subplots(figsize=(11, 4.5))
    ax.set_xlim(0, 11)
    ax.set_ylim(0, 4)
    ax.axis("off")
    boxes = [(1.2, "SQL Server\ndbo.Productos"), (4.5, "database/\nListProducts"), (7.5, "Scapegoat\nTree RAM"), (10.0, "Vue.js\nVisualizacion")]
    for i, (x, label) in enumerate(boxes):
        box = FancyBboxPatch((x - 1.0, 1.2), 2.0, 1.4, boxstyle="round,pad=0.05", fc="#eef4ff", ec="#3157d5", lw=2)
        ax.add_patch(box)
        ax.text(x, 1.9, label, ha="center", va="center", fontsize=9)
        if i < len(boxes) - 1:
            nx = boxes[i + 1][0]
            ax.annotate("", xy=(nx - 1.0, 1.9), xytext=(x + 1.0, 1.9), arrowprops=dict(arrowstyle="->", color="#475467", lw=2))
    ax.text(2.8, 1.0, "SELECT", ha="center", fontsize=8, color="#667085")
    ax.text(6.0, 1.0, "Insert(ID, Product)", ha="center", fontsize=8, color="#667085")
    ax.text(8.7, 1.0, "GET /api/tree", ha="center", fontsize=8, color="#667085")
    ax.set_title("Flujo de datos: persistencia a indice en memoria", fontsize=14, fontweight="bold")
    fig.savefig(OUT / "03-flujo-datos.png", dpi=180, bbox_inches="tight", facecolor="white")
    plt.close(fig)


def benchmark_bars():
    labels = ["Insert\n(1000)", "Search\n(n=100k)", "Delete\n(1000)"]
    ns_per_op = [1570757, 214, 1602745]
    colors = ["#3157d5", "#12b76a", "#d92d20"]
    fig, ax = plt.subplots(figsize=(8, 5))
    bars = ax.bar(labels, ns_per_op, color=colors, edgecolor="white", linewidth=1.2)
    ax.set_ylabel("nanosegundos / operacion (escala log)")
    ax.set_yscale("log")
    ax.set_title("Resultados Benchmark — Scapegoat Tree (alpha = 2/3)")
    for bar, val in zip(bars, ns_per_op):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() * 1.15, f"{val:,.0f}", ha="center", fontsize=9)
    fig.savefig(OUT / "04-benchmark.png", dpi=180, bbox_inches="tight", facecolor="white")
    plt.close(fig)


if __name__ == "__main__":
    architecture()
    insertion_flow()
    data_flow()
    benchmark_bars()
    print(f"Diagramas generados en {OUT}")
