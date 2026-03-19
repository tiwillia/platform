"""
Configuration loading for the Ambient Runner SDK.

Reads ambient.json, MCP server config, and repository configuration
from environment variables and the filesystem.
"""

import json as _json
import logging
import os
from pathlib import Path
from typing import Optional

from ambient_runner.platform.context import RunnerContext
from ambient_runner.platform.utils import expand_env_vars, parse_owner_repo

logger = logging.getLogger(__name__)


def load_ambient_config(cwd_path: str) -> dict:
    """Load ambient.json configuration from workflow directory.

    Returns:
        Parsed config dict, or empty dict if not found / invalid.
    """
    try:
        config_path = Path(cwd_path) / ".ambient" / "ambient.json"

        if not config_path.exists():
            logger.info(f"No ambient.json found at {config_path}, using defaults")
            return {}

        with open(config_path, "r") as f:
            config = _json.load(f)
            logger.info(f"Loaded ambient.json: name={config.get('name')}")
            return config

    except _json.JSONDecodeError as e:
        logger.error(f"Failed to parse ambient.json: {e}")
        return {}
    except Exception as e:
        logger.error(f"Error loading ambient.json: {e}")
        return {}


def _load_mcp_file(path: str) -> Optional[dict]:
    """Load and parse a single .mcp.json file, returning raw mcpServers dict.

    Returns:
        Dict of MCP server configs (not yet env-expanded), or None.
    """
    mcp_file = Path(path)
    if not mcp_file.exists() or not mcp_file.is_file():
        return None
    try:
        with open(mcp_file, "r") as f:
            config = _json.load(f)
        servers = config.get("mcpServers", {})
        if servers:
            logger.info(f"Loaded {len(servers)} MCP server(s) from {mcp_file}")
        return servers
    except _json.JSONDecodeError as e:
        logger.error(f"Failed to parse MCP config at {mcp_file}: {e}")
        return None
    except Exception as e:
        logger.error(f"Error loading MCP config at {mcp_file}: {e}")
        return None


def load_mcp_config(context: RunnerContext, cwd_path: str) -> Optional[dict]:
    """Load MCP server configuration from runner and project .mcp.json files.

    Reads the runner's built-in .mcp.json as a base, then merges any
    .mcp.json found in ``cwd_path`` (e.g. a workflow root) on top so
    that project-level servers can extend or override the defaults.

    Returns:
        Dict of MCP server configs with env vars expanded, or None.
    """
    # 1. Runner built-in .mcp.json (base)
    runner_path = context.get_env(
        "MCP_CONFIG_FILE", "/app/ambient-runner/.mcp.json"
    )
    merged = _load_mcp_file(runner_path) or {}

    # 2. Project / workflow .mcp.json (override by server name)
    cwd_mcp_path = str(Path(cwd_path) / ".mcp.json")
    if cwd_mcp_path != runner_path:
        project_servers = _load_mcp_file(cwd_mcp_path)
        if project_servers:
            merged.update(project_servers)
            logger.info(
                f"Merged {len(project_servers)} project MCP server(s) from {cwd_mcp_path}"
            )

    if not merged:
        logger.info("No MCP servers found in runner or project config")
        return None

    expanded = expand_env_vars(merged)
    logger.info(f"Expanded MCP config env vars for {len(expanded)} servers")
    return expanded


def get_user_mcp_servers() -> list[dict]:
    """Read user-defined MCP servers from MCP_SERVERS_JSON env var."""
    raw = os.environ.get("MCP_SERVERS_JSON", "").strip()
    if not raw:
        return []
    try:
        servers = _json.loads(raw)
        if isinstance(servers, list):
            logger.info(f"Loaded {len(servers)} user-defined MCP servers from MCP_SERVERS_JSON")
            return servers
        return []
    except _json.JSONDecodeError as e:
        logger.error(f"Failed to parse MCP_SERVERS_JSON: {e}")
        return []


def get_repos_config() -> list[dict]:
    """Read repos mapping from REPOS_JSON env if present.

    Expected format::

        [{"url": "...", "branch": "main", "autoPush": true}, ...]

    Returns:
        List of dicts: ``[{"name": ..., "url": ..., "branch": ..., "autoPush": bool}, ...]``
    """
    try:
        raw = os.getenv("REPOS_JSON", "").strip()
        if not raw:
            return []
        data = _json.loads(raw)
        if isinstance(data, list):
            out: list[dict] = []
            for it in data:
                if not isinstance(it, dict):
                    continue

                url = str(it.get("url") or "").strip()
                branch_from_json = it.get("branch")
                if branch_from_json and str(branch_from_json).strip():
                    branch = str(branch_from_json).strip()
                else:
                    session_id = os.getenv("AGENTIC_SESSION_NAME", "").strip()
                    branch = f"ambient/{session_id}" if session_id else "main"
                auto_push_raw = it.get("autoPush", False)
                auto_push = auto_push_raw if isinstance(auto_push_raw, bool) else False

                if not url:
                    continue

                name = str(it.get("name") or "").strip()
                if not name:
                    try:
                        _owner, repo, _ = parse_owner_repo(url)
                        derived = repo or ""
                        if not derived:
                            from urllib.parse import urlparse

                            p = urlparse(url)
                            parts = [pt for pt in (p.path or "").split("/") if pt]
                            if parts:
                                derived = parts[-1]
                        name = (derived or "").removesuffix(".git").strip()
                    except Exception:
                        name = ""

                if name and url:
                    out.append(
                        {
                            "name": name,
                            "url": url,
                            "branch": branch,
                            "autoPush": auto_push,
                        }
                    )
            return out
    except Exception:
        return []
    return []
