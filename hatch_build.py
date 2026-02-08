import platform
import subprocess
import sys
from pathlib import Path

try:
    import tomllib
except ImportError:
    import tomli as tomllib

from hatchling.builders.hooks.plugin.interface import BuildHookInterface


class CustomBuildHook(BuildHookInterface):
    def initialize(self, version, build_data):
        # Read version from pyproject.toml
        pyproject_path = Path(self.root) / "pyproject.toml"
        with open(pyproject_path, "rb") as f:
            pyproject_data = tomllib.load(f)
        project_version = pyproject_data["project"]["version"]

        system = platform.system()
        if system == "Windows":
            lib_name = "metastringedit.dll"
        elif system == "Darwin":
            lib_name = "metastringedit.dylib"
        else:  # Linux and others
            lib_name = "metastringedit.so"

        project_root = Path(self.root)
        go_src_dir = project_root / "binding"
        output_path = go_src_dir / lib_name

        print(
            f"\nBuilding Go shared library: {lib_name} with version {project_version}"
        )
        cmd = [
            "go",
            "build",
            "-o",
            str(output_path),
            "-buildmode=c-shared",
            f"-ldflags=-X metastringedit/cli.version={project_version}",
            ".",
        ]

        try:
            subprocess.run(
                cmd,
                cwd=go_src_dir,
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
            )
            print(f"Built {lib_name} successfully")
        except subprocess.CalledProcessError as e:
            print(f"Go build failed:\n{e.stdout}", file=sys.stderr)
            raise

        h_file = output_path.with_suffix(".h")
        if h_file.exists():
            h_file.unlink()

        build_data["pure_python"] = False

        os_name = sys.platform
        arch = platform.machine().lower()
        if arch in ["x86_64", "amd64"]:
            plat_arch = "win_amd64" if os_name == "win32" else "x86_64"
        elif arch in ["arm64", "aarch64"]:
            plat_arch = "arm64" if os_name == "win32" else "aarch64"
        else:
            plat_arch = arch

        if os_name == "win32":
            tag_platform = plat_arch
        elif os_name == "darwin":
            if arch == "arm64":
                tag_platform = "macosx_11_0_arm64"
            else:
                tag_platform = "macosx_10_9_x86_64"
        else:
            tag_platform = f"manylinux_2_17_{plat_arch}"

        build_data["tag"] = f"py3-none-{tag_platform}"
