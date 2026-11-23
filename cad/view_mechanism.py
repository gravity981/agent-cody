from cody_cad.mechanism import build_assembly_parts
from ocp_vscode import show

if __name__ == "__main__":
    parts = build_assembly_parts()
    names = list(parts)
    colors = ["#808080" if name == "GA12-N20 motor" else "#E8B024" for name in names]
    show(
        *parts.values(),
        names=names,
        colors=colors,
        up="Z",
        orbit_control=True,
        reset_camera="reset",
    )
