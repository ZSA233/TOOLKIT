from __future__ import annotations

import importlib.util
import sys
import tarfile
import tempfile
import textwrap
import unittest
import zipfile
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]


def load_releasectl():
    module_path = REPO_ROOT / "scripts" / "releasectl.py"
    spec = importlib.util.spec_from_file_location("releasectl", module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load releasectl module from {module_path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def write_release_tool(
    repo_root: Path,
    *,
    service_name: str,
    tool_id: str,
    version: str = "1.2.3",
    release_notes_path: str | None = None,
    targets_toml: str,
) -> None:
    release_notes_rel = release_notes_path or f"docs/zh/tools/{tool_id}.md"
    service_dir = repo_root / "services" / service_name
    release_file = service_dir / "release.toml"
    go_mod_file = service_dir / "go.mod"
    release_notes = repo_root / release_notes_rel

    go_mod_file.parent.mkdir(parents=True, exist_ok=True)
    go_mod_file.write_text(f"module {tool_id}\n", encoding="utf-8")
    release_notes.parent.mkdir(parents=True, exist_ok=True)
    release_notes.write_text(f"# {tool_id}\n", encoding="utf-8")

    release_file.parent.mkdir(parents=True, exist_ok=True)
    release_file.write_text(
        textwrap.dedent(
            f"""
            schema_version = 1

            [tool]
            id = "{tool_id}"
            name = "{tool_id}"
            version = "{version}"
            service_dir = "services/{service_name}"
            release_notes = "{release_notes_rel}"

            [github]
            tag_prefix = "{tool_id}/v"

            {targets_toml}
            """
        ).strip()
        + "\n",
        encoding="utf-8",
    )


class ReleaseCtlProjectTests(unittest.TestCase):
    def test_stable_and_rc_tags_are_derived_from_release_toml(self) -> None:
        releasectl = load_releasectl()

        release_tool = releasectl.load_release_tool("mtu-tuner", REPO_ROOT)

        self.assertEqual(release_tool.stable_tag, "mtu-tuner/v0.0.3")
        self.assertEqual(release_tool.rc_tag(2), "mtu-tuner/v0.0.3-rc.2")

    def test_release_matrix_keeps_gui_targets_native(self) -> None:
        releasectl = load_releasectl()

        matrix = releasectl.github_matrix("mtu-tuner", "release", REPO_ROOT)
        entries = {entry["id"]: entry for entry in matrix["include"]}

        self.assertEqual(entries["gui-linux-amd64"]["runner"], "ubuntu-24.04")
        self.assertEqual(entries["gui-linux-amd64"]["goos"], "linux")
        self.assertEqual(entries["gui-macos-arm64"]["runner"], "macos-14")
        self.assertEqual(entries["gui-macos-arm64"]["goos"], "darwin")
        self.assertEqual(entries["gui-windows-amd64"]["runner"], "windows-2022")
        self.assertEqual(entries["gui-windows-amd64"]["goos"], "windows")

    def test_release_matrix_no_longer_bundles_gui_dist_assets(self) -> None:
        releasectl = load_releasectl()

        matrix = releasectl.github_matrix("mtu-tuner", "release", REPO_ROOT)
        gui_entries = [entry for entry in matrix["include"] if entry["component"] == "gui"]

        self.assertTrue(gui_entries)
        self.assertTrue(all(entry["bundle_dist"] is False for entry in gui_entries))

    def test_release_matrix_uses_direct_binary_assets_for_mtu_tuner(self) -> None:
        releasectl = load_releasectl()

        matrix = releasectl.github_matrix("mtu-tuner", "release", REPO_ROOT)
        entries = {entry["id"]: entry for entry in matrix["include"]}

        self.assertTrue(all(entry["package_format"] == "binary" for entry in entries.values()))
        self.assertEqual(
            entries["cli-linux-amd64"]["asset_name"],
            "mtu-tuner_cli_linux_amd64",
        )
        self.assertEqual(
            entries["gui-macos-arm64"]["asset_name"],
            "mtu-tuner_gui_macos_arm64",
        )
        self.assertEqual(
            entries["gui-windows-amd64"]["asset_name"],
            "mtu-tuner_gui_windows_amd64.exe",
        )

    def test_find_release_tool_by_tag_matches_tag_prefix(self) -> None:
        releasectl = load_releasectl()

        release_tool = releasectl.load_release_tool_by_tag("mtu-tuner/v0.0.3-rc.2", REPO_ROOT)

        self.assertEqual(release_tool.id, "mtu-tuner")

    def test_host_matrix_filters_targets_by_goos(self) -> None:
        releasectl = load_releasectl()

        matrix = releasectl.host_matrix("mtu-tuner", "release", "windows", REPO_ROOT)

        self.assertEqual(
            {entry["id"] for entry in matrix["include"]},
            {"cli-windows-amd64", "gui-windows-amd64"},
        )


class ReleaseCtlDiscoveryTests(unittest.TestCase):
    def test_workflow_tools_include_ci_participants_with_short_test_commands(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            write_release_tool(
                repo_root,
                service_name="alpha_tool",
                tool_id="alpha-tool",
                targets_toml=textwrap.dedent(
                    """
                    [[target]]
                    id = "cli-linux-amd64"
                    component = "cli"
                    runner = "ubuntu-24.04"
                    platform = "linux"
                    arch = "amd64"
                    goos = "linux"
                    goarch = "amd64"
                    package_format = "tar.gz"
                    workflows = ["ci", "release"]
                    """
                ).strip(),
            )
            write_release_tool(
                repo_root,
                service_name="beta_tool",
                tool_id="beta-tool",
                targets_toml=textwrap.dedent(
                    """
                    [[target]]
                    id = "cli-linux-amd64"
                    component = "cli"
                    runner = "ubuntu-24.04"
                    platform = "linux"
                    arch = "amd64"
                    goos = "linux"
                    goarch = "amd64"
                    package_format = "tar.gz"
                    workflows = ["release"]
                    """
                ).strip(),
            )

            matrix = releasectl.workflow_tools("ci", repo_root)

            self.assertEqual(
                matrix,
                {
                    "include": [
                        {
                            "tool": "alpha-tool",
                            "runner": "ubuntu-24.04",
                            "go_mod_file": "services/alpha_tool/go.mod",
                            "needs_linux_gui_deps": False,
                            "test_command": "make alpha-tool-test GO_TEST_FLAGS=-short",
                            "validate_command": "python3 scripts/releasectl.py validate --tool alpha-tool --workflow ci",
                        }
                    ]
                },
            )

    def test_workflow_tools_require_linux_gui_deps_when_ci_targets_include_gui_linux(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            write_release_tool(
                repo_root,
                service_name="alpha_tool",
                tool_id="alpha-tool",
                targets_toml=textwrap.dedent(
                    """
                    [[target]]
                    id = "cli-linux-amd64"
                    component = "cli"
                    runner = "ubuntu-24.04"
                    platform = "linux"
                    arch = "amd64"
                    goos = "linux"
                    goarch = "amd64"
                    package_format = "tar.gz"
                    workflows = ["ci", "release"]

                    [[target]]
                    id = "gui-linux-amd64"
                    component = "gui"
                    runner = "ubuntu-24.04"
                    platform = "linux"
                    arch = "amd64"
                    goos = "linux"
                    goarch = "amd64"
                    package_format = "tar.gz"
                    bundle_dist = true
                    workflows = ["ci", "release"]
                    """
                ).strip(),
            )

            matrix = releasectl.workflow_tools("ci", repo_root)

            self.assertEqual(len(matrix["include"]), 1)
            self.assertTrue(matrix["include"][0]["needs_linux_gui_deps"])

    def test_workflow_matrix_flattens_targets_across_tools(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            write_release_tool(
                repo_root,
                service_name="alpha_tool",
                tool_id="alpha-tool",
                targets_toml=textwrap.dedent(
                    """
                    [[target]]
                    id = "cli-linux-amd64"
                    component = "cli"
                    runner = "ubuntu-24.04"
                    platform = "linux"
                    arch = "amd64"
                    goos = "linux"
                    goarch = "amd64"
                    package_format = "tar.gz"
                    workflows = ["ci"]
                    """
                ).strip(),
            )
            write_release_tool(
                repo_root,
                service_name="beta_tool",
                tool_id="beta-tool",
                targets_toml=textwrap.dedent(
                    """
                    [[target]]
                    id = "gui-macos-arm64"
                    component = "gui"
                    runner = "macos-14"
                    platform = "macos"
                    arch = "arm64"
                    goos = "darwin"
                    goarch = "arm64"
                    package_format = "tar.gz"
                    bundle_dist = true
                    workflows = ["ci"]
                    """
                ).strip(),
            )

            matrix = releasectl.workflow_matrix("ci", repo_root)

            self.assertEqual(
                [(entry["tool"], entry["id"], entry["runner"]) for entry in matrix["include"]],
                [
                    ("alpha-tool", "cli-linux-amd64", "ubuntu-24.04"),
                    ("beta-tool", "gui-macos-arm64", "macos-14"),
                ],
            )

    def test_discover_release_tools_rejects_duplicate_tool_ids(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            targets_toml = textwrap.dedent(
                """
                [[target]]
                id = "cli-linux-amd64"
                component = "cli"
                runner = "ubuntu-24.04"
                platform = "linux"
                arch = "amd64"
                goos = "linux"
                goarch = "amd64"
                package_format = "tar.gz"
                workflows = ["ci"]
                """
            ).strip()
            write_release_tool(repo_root, service_name="alpha_tool", tool_id="dup-tool", targets_toml=targets_toml)
            write_release_tool(repo_root, service_name="beta_tool", tool_id="dup-tool", targets_toml=targets_toml)

            with self.assertRaisesRegex(releasectl.ReleaseCtlError, "duplicate tool id 'dup-tool'"):
                releasectl.discover_release_tools(repo_root)


class ReleaseCtlPackagingTests(unittest.TestCase):
    def test_package_cli_target_creates_tarball_with_public_binary_name(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            release_file = repo_root / "services" / "demo_tool" / "release.toml"
            go_mod_file = repo_root / "services" / "demo_tool" / "go.mod"
            release_notes = repo_root / "docs" / "zh" / "tools" / "demo.md"
            binary_path = repo_root / "services" / "demo_tool" / "build" / "bin" / "cli" / "linux_amd64" / "demo-tool"
            binary_path.parent.mkdir(parents=True, exist_ok=True)
            binary_path.write_text("cli", encoding="utf-8")
            go_mod_file.parent.mkdir(parents=True, exist_ok=True)
            go_mod_file.write_text("module demo-tool\n", encoding="utf-8")
            release_notes.parent.mkdir(parents=True, exist_ok=True)
            release_notes.write_text("# Demo\n", encoding="utf-8")

            release_file.parent.mkdir(parents=True, exist_ok=True)
            release_file.write_text(
                textwrap.dedent(
                    """
                    schema_version = 1

                    [tool]
                    id = "demo-tool"
                    name = "demo-tool"
                    version = "1.2.3"
                    service_dir = "services/demo_tool"
                    release_notes = "docs/zh/tools/demo.md"

                    [github]
                    tag_prefix = "demo-tool/v"

                    [[target]]
                    id = "cli-linux-amd64"
                    component = "cli"
                    runner = "ubuntu-24.04"
                    platform = "linux"
                    arch = "amd64"
                    goos = "linux"
                    goarch = "amd64"
                    package_format = "tar.gz"
                    workflows = ["release"]
                    """
                ).strip()
                + "\n",
                encoding="utf-8",
            )

            archive_path = releasectl.package_target("demo-tool", "cli-linux-amd64", repo_root)

            with tarfile.open(archive_path, "r:gz") as archive:
                self.assertIn("demo-tool/demo-tool", archive.getnames())

    def test_package_gui_target_copies_dist_assets_into_zip(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            release_file = repo_root / "services" / "demo_tool" / "release.toml"
            go_mod_file = repo_root / "services" / "demo_tool" / "go.mod"
            release_notes = repo_root / "docs" / "zh" / "tools" / "demo.md"
            binary_path = repo_root / "services" / "demo_tool" / "build" / "bin" / "gui" / "windows_amd64" / "demo-tool.exe"
            dist_asset_path = repo_root / "services" / "demo_tool" / "build" / "bin" / "gui" / "windows_amd64" / "dist" / "index.html"
            binary_path.parent.mkdir(parents=True, exist_ok=True)
            binary_path.write_text("gui", encoding="utf-8")
            dist_asset_path.parent.mkdir(parents=True, exist_ok=True)
            dist_asset_path.write_text("<html></html>", encoding="utf-8")
            go_mod_file.parent.mkdir(parents=True, exist_ok=True)
            go_mod_file.write_text("module demo-tool\n", encoding="utf-8")
            release_notes.parent.mkdir(parents=True, exist_ok=True)
            release_notes.write_text("# Demo\n", encoding="utf-8")

            release_file.parent.mkdir(parents=True, exist_ok=True)
            release_file.write_text(
                textwrap.dedent(
                    """
                    schema_version = 1

                    [tool]
                    id = "demo-tool"
                    name = "demo-tool"
                    version = "1.2.3"
                    service_dir = "services/demo_tool"
                    release_notes = "docs/zh/tools/demo.md"

                    [github]
                    tag_prefix = "demo-tool/v"

                    [[target]]
                    id = "gui-windows-amd64"
                    component = "gui"
                    runner = "windows-2022"
                    platform = "windows"
                    arch = "amd64"
                    goos = "windows"
                    goarch = "amd64"
                    package_format = "zip"
                    bundle_dist = true
                    workflows = ["release"]
                    """
                ).strip()
                + "\n",
                encoding="utf-8",
            )

            archive_path = releasectl.package_target("demo-tool", "gui-windows-amd64", repo_root)

            with zipfile.ZipFile(archive_path) as archive:
                self.assertIn("demo-tool/demo-tool.exe", archive.namelist())
                self.assertIn("demo-tool/dist/index.html", archive.namelist())

    def test_package_binary_target_copies_release_asset_without_archive_wrapper(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            release_file = repo_root / "services" / "demo_tool" / "release.toml"
            go_mod_file = repo_root / "services" / "demo_tool" / "go.mod"
            release_notes = repo_root / "docs" / "zh" / "tools" / "demo.md"
            binary_path = repo_root / "services" / "demo_tool" / "build" / "bin" / "gui" / "windows_amd64" / "demo-tool.exe"
            binary_path.parent.mkdir(parents=True, exist_ok=True)
            binary_path.write_text("gui", encoding="utf-8")
            go_mod_file.parent.mkdir(parents=True, exist_ok=True)
            go_mod_file.write_text("module demo-tool\n", encoding="utf-8")
            release_notes.parent.mkdir(parents=True, exist_ok=True)
            release_notes.write_text("# Demo\n", encoding="utf-8")

            release_file.parent.mkdir(parents=True, exist_ok=True)
            release_file.write_text(
                textwrap.dedent(
                    """
                    schema_version = 1

                    [tool]
                    id = "demo-tool"
                    name = "demo-tool"
                    version = "1.2.3"
                    service_dir = "services/demo_tool"
                    release_notes = "docs/zh/tools/demo.md"

                    [github]
                    tag_prefix = "demo-tool/v"

                    [[target]]
                    id = "gui-windows-amd64"
                    component = "gui"
                    runner = "windows-2022"
                    platform = "windows"
                    arch = "amd64"
                    goos = "windows"
                    goarch = "amd64"
                    package_format = "binary"
                    workflows = ["release"]
                    """
                ).strip()
                + "\n",
                encoding="utf-8",
            )

            asset_path = releasectl.package_target("demo-tool", "gui-windows-amd64", repo_root)

            self.assertEqual(asset_path.name, "demo-tool_gui_windows_amd64.exe")
            self.assertEqual(asset_path.read_text(encoding="utf-8"), "gui")

    def test_package_binary_target_removes_stale_archive_outputs(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            release_file = repo_root / "services" / "demo_tool" / "release.toml"
            go_mod_file = repo_root / "services" / "demo_tool" / "go.mod"
            release_notes = repo_root / "docs" / "zh" / "tools" / "demo.md"
            binary_path = repo_root / "services" / "demo_tool" / "build" / "bin" / "gui" / "windows_amd64" / "demo-tool.exe"
            release_root = repo_root / "services" / "demo_tool" / "build" / "release" / "gui" / "windows_amd64"
            stale_archive_path = release_root / "demo-tool_gui_windows_amd64.zip"
            stale_bundle_path = release_root / "demo-tool"
            binary_path.parent.mkdir(parents=True, exist_ok=True)
            binary_path.write_text("gui", encoding="utf-8")
            stale_bundle_path.mkdir(parents=True, exist_ok=True)
            stale_archive_path.parent.mkdir(parents=True, exist_ok=True)
            stale_archive_path.write_text("old", encoding="utf-8")
            (stale_bundle_path / "demo-tool.exe").write_text("old", encoding="utf-8")
            go_mod_file.parent.mkdir(parents=True, exist_ok=True)
            go_mod_file.write_text("module demo-tool\n", encoding="utf-8")
            release_notes.parent.mkdir(parents=True, exist_ok=True)
            release_notes.write_text("# Demo\n", encoding="utf-8")

            release_file.parent.mkdir(parents=True, exist_ok=True)
            release_file.write_text(
                textwrap.dedent(
                    """
                    schema_version = 1

                    [tool]
                    id = "demo-tool"
                    name = "demo-tool"
                    version = "1.2.3"
                    service_dir = "services/demo_tool"
                    release_notes = "docs/zh/tools/demo.md"

                    [github]
                    tag_prefix = "demo-tool/v"

                    [[target]]
                    id = "gui-windows-amd64"
                    component = "gui"
                    runner = "windows-2022"
                    platform = "windows"
                    arch = "amd64"
                    goos = "windows"
                    goarch = "amd64"
                    package_format = "binary"
                    workflows = ["release"]
                    """
                ).strip()
                + "\n",
                encoding="utf-8",
            )

            asset_path = releasectl.package_target("demo-tool", "gui-windows-amd64", repo_root)

            self.assertTrue(asset_path.exists())
            self.assertFalse(stale_archive_path.exists())
            self.assertFalse(stale_bundle_path.exists())

    def test_validate_rejects_binary_format_when_bundle_dist_is_enabled(self) -> None:
        releasectl = load_releasectl()

        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = Path(tmpdir)
            write_release_tool(
                repo_root,
                service_name="demo_tool",
                tool_id="demo-tool",
                targets_toml=textwrap.dedent(
                    """
                    [[target]]
                    id = "gui-linux-amd64"
                    component = "gui"
                    runner = "ubuntu-24.04"
                    platform = "linux"
                    arch = "amd64"
                    goos = "linux"
                    goarch = "amd64"
                    package_format = "binary"
                    bundle_dist = true
                    workflows = ["release"]
                    """
                ).strip(),
            )

            with self.assertRaisesRegex(
                releasectl.ReleaseCtlError,
                "cannot set bundle_dist when package_format is 'binary'",
            ):
                releasectl.load_release_tool("demo-tool", repo_root)


if __name__ == "__main__":
    unittest.main()
