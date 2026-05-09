#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import shutil
import sys
import tarfile
import zipfile
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import tomllib


SEMVER_RE = re.compile(r"^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$")
TOOL_ID_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
ALLOWED_WORKFLOWS = {"ci", "release", "release-rc"}
RUNNER_PLATFORM_PREFIX = {
    "linux": "ubuntu-",
    "macos": "macos-",
    "windows": "windows-",
}
GOOS_TO_PLATFORM = {
    "linux": "linux",
    "darwin": "macos",
    "windows": "windows",
}


class ReleaseCtlError(RuntimeError):
    pass


@dataclass(frozen=True)
class BuildTarget:
    id: str
    component: str
    runner: str
    platform: str
    arch: str
    goos: str
    goarch: str
    package_format: str
    workflows: tuple[str, ...]
    bundle_dist: bool = False

    def public_binary_name(self, tool_id: str) -> str:
        return f"{tool_id}.exe" if self.goos == "windows" else tool_id

    @property
    def platform_dir(self) -> str:
        return f"{self.goos}_{self.goarch}"

    @property
    def build_subdir(self) -> str:
        return "cli" if self.component == "cli" else "gui"

    def build_target(self, tool_id: str) -> str:
        return f"{tool_id}-build" if self.component == "cli" else f"{tool_id}-gui-build"

    def build_vars(self, tool_id: str) -> dict[str, str]:
        build_vars = {"GOOS": self.goos, "GOARCH": self.goarch}
        if self.component == "cli":
            build_vars["CMD"] = "cli"
            build_vars["BIN_NAME"] = tool_id
        else:
            build_vars["GUI_BIN_NAME"] = tool_id
        return build_vars

    def build_command(self, tool_id: str) -> str:
        vars_str = " ".join(f"{key}={value}" for key, value in self.build_vars(tool_id).items())
        return f"make {self.build_target(tool_id)} {vars_str}"


@dataclass(frozen=True)
class ReleaseTool:
    id: str
    name: str
    version: str
    service_dir: Path
    release_notes: Path
    tag_prefix: str
    targets: tuple[BuildTarget, ...]

    @property
    def stable_tag(self) -> str:
        return f"{self.tag_prefix}{self.version}"

    @property
    def go_mod_file(self) -> Path:
        return self.service_dir / "go.mod"

    @property
    def frontend_lock_file(self) -> Path:
        return self.service_dir / "cmd" / "gui" / "frontend" / "pnpm-lock.yaml"

    def rc_tag(self, rc_number: int) -> str:
        if rc_number <= 0:
            raise ReleaseCtlError("rc number must be positive")
        return f"{self.stable_tag}-rc.{rc_number}"

    def validate_tag(self, tag: str) -> dict[str, Any]:
        pattern = re.compile(rf"^{re.escape(self.tag_prefix)}{re.escape(self.version)}(?:-rc\.(\d+))?$")
        match = pattern.fullmatch(tag)
        if match is None:
            raise ReleaseCtlError(
                f"tag {tag!r} does not match release version {self.version!r} with prefix {self.tag_prefix!r}"
            )
        rc_text = match.group(1)
        return {
            "tag": tag,
            "version": self.version,
            "prerelease": rc_text is not None,
            "rc": int(rc_text) if rc_text is not None else None,
        }

    def target(self, target_id: str) -> BuildTarget:
        for target in self.targets:
            if target.id == target_id:
                return target
        raise ReleaseCtlError(f"unknown target {target_id!r} for tool {self.id!r}")


def repo_root_from_script() -> Path:
    return Path(__file__).resolve().parents[1]


def release_files(repo_root: Path) -> list[Path]:
    return sorted(repo_root.glob("services/*/release.toml"))


def load_release_tool(tool_id: str, repo_root: Path | str | None = None) -> ReleaseTool:
    resolved_root = Path(repo_root) if repo_root is not None else repo_root_from_script()
    matches: list[Path] = []
    for release_file in release_files(resolved_root):
        data = load_toml(release_file)
        if data.get("tool", {}).get("id") == tool_id:
            matches.append(release_file)
    if not matches:
        raise ReleaseCtlError(f"no release.toml found for tool {tool_id!r}")
    if len(matches) > 1:
        raise ReleaseCtlError(f"multiple release.toml files found for tool {tool_id!r}: {matches}")
    return parse_release_tool(matches[0], resolved_root)


def load_release_tool_by_tag(tag: str, repo_root: Path | str | None = None) -> ReleaseTool:
    resolved_root = Path(repo_root) if repo_root is not None else repo_root_from_script()
    matches: list[ReleaseTool] = []
    for release_file in release_files(resolved_root):
        release_tool = parse_release_tool(release_file, resolved_root)
        if tag.startswith(release_tool.tag_prefix):
            matches.append(release_tool)
    if not matches:
        raise ReleaseCtlError(f"no release.toml matches tag {tag!r}")
    if len(matches) > 1:
        matched_ids = ", ".join(tool.id for tool in matches)
        raise ReleaseCtlError(f"multiple tools match tag {tag!r}: {matched_ids}")
    return matches[0]


def load_toml(path: Path) -> dict[str, Any]:
    with path.open("rb") as handle:
        return tomllib.load(handle)


def discover_release_tools(repo_root: Path | str | None = None) -> tuple[ReleaseTool, ...]:
    resolved_root = Path(repo_root) if repo_root is not None else repo_root_from_script()
    tools: list[ReleaseTool] = []
    seen_ids: dict[str, Path] = {}
    for release_file in release_files(resolved_root):
        tool = parse_release_tool(release_file, resolved_root)
        if tool.id in seen_ids:
            raise ReleaseCtlError(
                f"duplicate tool id {tool.id!r} in {seen_ids[tool.id]} and {release_file}"
            )
        seen_ids[tool.id] = release_file
        tools.append(tool)
    return tuple(tools)


def parse_release_tool(release_file: Path, repo_root: Path) -> ReleaseTool:
    data = load_toml(release_file)
    if data.get("schema_version") != 1:
        raise ReleaseCtlError(f"{release_file} must set schema_version = 1")

    tool_data = data.get("tool")
    github_data = data.get("github")
    target_data = data.get("target")
    if not isinstance(tool_data, dict):
        raise ReleaseCtlError(f"{release_file} is missing [tool]")
    if not isinstance(github_data, dict):
        raise ReleaseCtlError(f"{release_file} is missing [github]")
    if not isinstance(target_data, list) or not target_data:
        raise ReleaseCtlError(f"{release_file} must define at least one [[target]]")

    tool_id = require_string(tool_data, "id", release_file)
    name = require_string(tool_data, "name", release_file)
    version = require_string(tool_data, "version", release_file)
    service_dir = Path(require_string(tool_data, "service_dir", release_file))
    release_notes = Path(require_string(tool_data, "release_notes", release_file))
    tag_prefix = require_string(github_data, "tag_prefix", release_file)

    targets = tuple(parse_target(item, release_file) for item in target_data)
    tool = ReleaseTool(
        id=tool_id,
        name=name,
        version=version,
        service_dir=service_dir,
        release_notes=release_notes,
        tag_prefix=tag_prefix,
        targets=targets,
    )
    validate_release_tool(tool, repo_root)
    return tool


def require_string(mapping: dict[str, Any], key: str, release_file: Path) -> str:
    value = mapping.get(key)
    if not isinstance(value, str) or not value:
        raise ReleaseCtlError(f"{release_file} field {key!r} must be a non-empty string")
    return value


def parse_target(item: dict[str, Any], release_file: Path) -> BuildTarget:
    workflows = item.get("workflows")
    if not isinstance(workflows, list) or not workflows or not all(isinstance(entry, str) for entry in workflows):
        raise ReleaseCtlError(f"{release_file} target workflows must be a non-empty string list")
    bundle_dist = item.get("bundle_dist", False)
    if not isinstance(bundle_dist, bool):
        raise ReleaseCtlError(f"{release_file} target bundle_dist must be a boolean")
    return BuildTarget(
        id=require_string(item, "id", release_file),
        component=require_string(item, "component", release_file),
        runner=require_string(item, "runner", release_file),
        platform=require_string(item, "platform", release_file),
        arch=require_string(item, "arch", release_file),
        goos=require_string(item, "goos", release_file),
        goarch=require_string(item, "goarch", release_file),
        package_format=require_string(item, "package_format", release_file),
        workflows=tuple(workflows),
        bundle_dist=bundle_dist,
    )


def validate_release_tool(tool: ReleaseTool, repo_root: Path) -> None:
    if not TOOL_ID_RE.fullmatch(tool.id):
        raise ReleaseCtlError(f"tool id {tool.id!r} must be kebab-case")
    if tool.name != tool.id:
        raise ReleaseCtlError(f"tool name {tool.name!r} must match tool id {tool.id!r} for public naming")
    if not SEMVER_RE.fullmatch(tool.version):
        raise ReleaseCtlError(f"version {tool.version!r} must be X.Y.Z")
    if tool.tag_prefix != f"{tool.id}/v":
        raise ReleaseCtlError(f"tag_prefix {tool.tag_prefix!r} must be {tool.id + '/v'!r}")
    if not (repo_root / tool.service_dir).exists():
        raise ReleaseCtlError(f"service_dir {tool.service_dir} does not exist")
    if not (repo_root / tool.release_notes).exists():
        raise ReleaseCtlError(f"release_notes {tool.release_notes} does not exist")
    if not (repo_root / tool.go_mod_file).exists():
        raise ReleaseCtlError(f"go_mod_file {tool.go_mod_file} does not exist")

    seen_ids: set[str] = set()
    for target in tool.targets:
        if target.id in seen_ids:
            raise ReleaseCtlError(f"duplicate target id {target.id!r}")
        seen_ids.add(target.id)
        validate_target(target)


def validate_target(target: BuildTarget) -> None:
    if target.component not in {"cli", "gui"}:
        raise ReleaseCtlError(f"target {target.id!r} has unsupported component {target.component!r}")
    if target.platform not in RUNNER_PLATFORM_PREFIX:
        raise ReleaseCtlError(f"target {target.id!r} has unsupported platform {target.platform!r}")
    if target.goos not in GOOS_TO_PLATFORM:
        raise ReleaseCtlError(f"target {target.id!r} has unsupported goos {target.goos!r}")
    if GOOS_TO_PLATFORM[target.goos] != target.platform:
        raise ReleaseCtlError(
            f"target {target.id!r} goos {target.goos!r} must match platform {target.platform!r}"
        )
    if target.goarch != target.arch:
        raise ReleaseCtlError(f"target {target.id!r} goarch {target.goarch!r} must match arch {target.arch!r}")
    if target.package_format not in {"tar.gz", "zip"}:
        raise ReleaseCtlError(f"target {target.id!r} has unsupported package_format {target.package_format!r}")
    if not set(target.workflows).issubset(ALLOWED_WORKFLOWS):
        raise ReleaseCtlError(f"target {target.id!r} has unsupported workflows {target.workflows!r}")
    if target.component == "gui" and not target.runner.startswith(RUNNER_PLATFORM_PREFIX[target.platform]):
        raise ReleaseCtlError(
            f"target {target.id!r} must use a native runner for GUI builds, got {target.runner!r} for {target.platform!r}"
        )
    if target.component == "cli" and target.bundle_dist:
        raise ReleaseCtlError(f"target {target.id!r} cannot set bundle_dist for a CLI release")


def metadata(tool_id: str, repo_root: Path | str | None = None, tag: str | None = None) -> dict[str, Any]:
    release_tool = load_release_tool(tool_id, repo_root)
    tag_info = release_tool.validate_tag(tag) if tag is not None else release_tool.validate_tag(release_tool.stable_tag)
    return {
        "tool": release_tool.id,
        "version": release_tool.version,
        "tag": tag_info["tag"],
        "prerelease": tag_info["prerelease"],
        "rc": tag_info["rc"],
        "release_name": tag_info["tag"],
        "release_notes": release_tool.release_notes.as_posix(),
        "service_dir": release_tool.service_dir.as_posix(),
        "go_mod_file": release_tool.go_mod_file.as_posix(),
        "frontend_lock_file": release_tool.frontend_lock_file.as_posix(),
    }


def workflow_targets(release_tool: ReleaseTool, workflow: str) -> tuple[BuildTarget, ...]:
    if workflow not in ALLOWED_WORKFLOWS:
        raise ReleaseCtlError(f"workflow must be one of {sorted(ALLOWED_WORKFLOWS)}")
    return tuple(target for target in release_tool.targets if workflow in target.workflows)


def github_matrix(tool_id: str, workflow: str, repo_root: Path | str | None = None) -> dict[str, Any]:
    release_tool = load_release_tool(tool_id, repo_root)
    include = [target_matrix_entry(release_tool, target) for target in workflow_targets(release_tool, workflow)]
    if not include:
        raise ReleaseCtlError(f"workflow {workflow!r} has no targets for tool {tool_id!r}")
    return {"include": include}


def workflow_tools(workflow: str, repo_root: Path | str | None = None) -> dict[str, Any]:
    include = []
    for release_tool in discover_release_tools(repo_root):
        targets = workflow_targets(release_tool, workflow)
        if not targets:
            continue
        include.append(workflow_tool_entry(release_tool, workflow, targets))
    if not include:
        raise ReleaseCtlError(f"workflow {workflow!r} has no tools")
    return {"include": include}


def workflow_matrix(workflow: str, repo_root: Path | str | None = None) -> dict[str, Any]:
    include = []
    for release_tool in discover_release_tools(repo_root):
        include.extend(target_matrix_entry(release_tool, target) for target in workflow_targets(release_tool, workflow))
    if not include:
        raise ReleaseCtlError(f"workflow {workflow!r} has no targets")
    return {"include": include}


def host_matrix(tool_id: str, workflow: str, host_goos: str, repo_root: Path | str | None = None) -> dict[str, Any]:
    include = [
        entry
        for entry in github_matrix(tool_id, workflow, repo_root)["include"]
        if entry["goos"] == host_goos
    ]
    if not include:
        raise ReleaseCtlError(f"workflow {workflow!r} has no targets for tool {tool_id!r} on host {host_goos!r}")
    return {"include": include}


def target_matrix_entry(release_tool: ReleaseTool, target: BuildTarget) -> dict[str, Any]:
    build_root = release_tool.service_dir / "build" / "bin" / target.build_subdir / target.platform_dir
    release_root = release_tool.service_dir / "build" / "release" / target.build_subdir / target.platform_dir
    archive_name = f"{release_tool.id}_{target.component}_{target.platform}_{target.arch}.{target.package_format}"
    return {
        "tool": release_tool.id,
        "id": target.id,
        "component": target.component,
        "runner": target.runner,
        "platform": target.platform,
        "arch": target.arch,
        "goos": target.goos,
        "goarch": target.goarch,
        "build_command": target.build_command(release_tool.id),
        "package_command": f"python3 scripts/releasectl.py package --tool {release_tool.id} --target {target.id}",
        "binary_path": (build_root / target.public_binary_name(release_tool.id)).as_posix(),
        "dist_path": (build_root / "dist").as_posix(),
        "bundle_dir": (release_root / release_tool.id).as_posix(),
        "archive_path": (release_root / archive_name).as_posix(),
        "archive_name": archive_name,
        "artifact_name": archive_name,
        "package_format": target.package_format,
        "bundle_dist": target.bundle_dist,
        "needs_linux_gui_deps": target.component == "gui" and target.goos == "linux",
        "go_mod_file": release_tool.go_mod_file.as_posix(),
        "frontend_lock_file": release_tool.frontend_lock_file.as_posix(),
    }


def workflow_tool_entry(release_tool: ReleaseTool, workflow: str, targets: list[BuildTarget]) -> dict[str, Any]:
    # CI discovery only knows about release metadata, so tools opt in by exposing
    # the standard <tool>-test make target alongside release.toml.
    verify_target = targets[0]
    return {
        "tool": release_tool.id,
        "runner": verify_target.runner,
        "go_mod_file": release_tool.go_mod_file.as_posix(),
        "needs_linux_gui_deps": any(target.component == "gui" and target.goos == "linux" for target in targets),
        "test_command": f"make {release_tool.id}-test GO_TEST_FLAGS=-short",
        "validate_command": f"python3 scripts/releasectl.py validate --tool {release_tool.id} --workflow {workflow}",
    }


def package_target(tool_id: str, target_id: str, repo_root: Path | str | None = None) -> Path:
    resolved_root = Path(repo_root) if repo_root is not None else repo_root_from_script()
    release_tool = load_release_tool(tool_id, resolved_root)
    target = release_tool.target(target_id)
    target_entry = target_matrix_entry(release_tool, target)

    binary_path = resolved_root / target_entry["binary_path"]
    dist_path = resolved_root / target_entry["dist_path"]
    bundle_dir = resolved_root / target_entry["bundle_dir"]
    archive_path = resolved_root / target_entry["archive_path"]
    stage_root = archive_path.parent

    if not binary_path.exists():
        raise ReleaseCtlError(f"missing binary for target {target_id!r}: {binary_path}")
    if target.bundle_dist and not dist_path.exists():
        raise ReleaseCtlError(f"missing GUI dist assets for target {target_id!r}: {dist_path}")

    if bundle_dir.exists():
        shutil.rmtree(bundle_dir)
    bundle_dir.mkdir(parents=True, exist_ok=True)
    shutil.copy2(binary_path, bundle_dir / target.public_binary_name(release_tool.id))
    if target.bundle_dist:
        shutil.copytree(dist_path, bundle_dir / "dist")

    if archive_path.exists():
        archive_path.unlink()
    archive_path.parent.mkdir(parents=True, exist_ok=True)
    if target.package_format == "zip":
        write_zip_archive(bundle_dir, archive_path)
    else:
        write_tar_archive(bundle_dir, archive_path)
    return archive_path


def write_zip_archive(bundle_dir: Path, archive_path: Path) -> None:
    with zipfile.ZipFile(archive_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for path in sorted(bundle_dir.rglob("*")):
            archive.write(path, path.relative_to(bundle_dir.parent))


def write_tar_archive(bundle_dir: Path, archive_path: Path) -> None:
    with tarfile.open(archive_path, "w:gz") as archive:
        archive.add(bundle_dir, arcname=bundle_dir.name)


def make_output(value: Any) -> str:
    if isinstance(value, str):
        return value
    return json.dumps(value, sort_keys=True)


def workflow_for_tag(tag: str) -> str:
    if "-rc." in tag:
        return "release-rc"
    if "/v" in tag:
        return "release"
    raise ReleaseCtlError(f"tag {tag!r} is not a supported release tag")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Release metadata helper for toolkit tools.")
    parser.add_argument("--repo-root", default=str(repo_root_from_script()), help="repository root path")
    subparsers = parser.add_subparsers(dest="command", required=True)

    for command_name in (
        "validate",
        "version",
        "tag",
        "metadata",
        "github-matrix",
        "workflow-tools",
        "workflow-matrix",
        "host-matrix",
        "package",
        "tool-from-tag",
        "workflow-from-tag",
    ):
        subparser = subparsers.add_parser(command_name)
        if command_name not in {"tool-from-tag", "workflow-from-tag", "workflow-tools", "workflow-matrix"}:
            subparser.add_argument("--tool", required=True)
        if command_name in {"validate", "metadata"}:
            subparser.add_argument("--tag")
        if command_name == "tag":
            subparser.add_argument("--rc", type=int)
        if command_name in {"github-matrix", "workflow-tools", "workflow-matrix"}:
            subparser.add_argument("--workflow", required=True)
        if command_name == "host-matrix":
            subparser.add_argument("--workflow", required=True)
            subparser.add_argument("--goos", required=True)
        if command_name == "validate":
            subparser.add_argument("--workflow")
        if command_name == "package":
            subparser.add_argument("--target", required=True)
        if command_name in {"tool-from-tag", "workflow-from-tag"}:
            subparser.add_argument("--tag", required=True)
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    repo_root = Path(args.repo_root)
    try:
        if args.command == "validate":
            release_tool = load_release_tool(args.tool, repo_root)
            if args.tag:
                release_tool.validate_tag(args.tag)
            if args.workflow:
                github_matrix(args.tool, args.workflow, repo_root)
            print(make_output({"tool": release_tool.id, "version": release_tool.version, "stable_tag": release_tool.stable_tag}))
            return 0
        if args.command == "version":
            print(load_release_tool(args.tool, repo_root).version)
            return 0
        if args.command == "tag":
            release_tool = load_release_tool(args.tool, repo_root)
            print(release_tool.rc_tag(args.rc) if args.rc else release_tool.stable_tag)
            return 0
        if args.command == "metadata":
            print(make_output(metadata(args.tool, repo_root, args.tag)))
            return 0
        if args.command == "github-matrix":
            print(make_output(github_matrix(args.tool, args.workflow, repo_root)))
            return 0
        if args.command == "workflow-tools":
            print(make_output(workflow_tools(args.workflow, repo_root)))
            return 0
        if args.command == "workflow-matrix":
            print(make_output(workflow_matrix(args.workflow, repo_root)))
            return 0
        if args.command == "host-matrix":
            print(make_output(host_matrix(args.tool, args.workflow, args.goos, repo_root)))
            return 0
        if args.command == "package":
            archive_path = package_target(args.tool, args.target, repo_root)
            print(make_output({"archive_path": archive_path.as_posix()}))
            return 0
        if args.command == "tool-from-tag":
            print(load_release_tool_by_tag(args.tag, repo_root).id)
            return 0
        if args.command == "workflow-from-tag":
            print(workflow_for_tag(args.tag))
            return 0
    except ReleaseCtlError as exc:
        print(f"releasectl: {exc}", file=sys.stderr)
        return 1
    parser.error(f"unsupported command {args.command!r}")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
