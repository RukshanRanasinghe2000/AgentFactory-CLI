import json
import os
import re
import subprocess
import requests


# ── Helpers ───────────────────────────────────────────────────────────────────

def _resolve_env(value: str) -> str:
    """Resolve ${env:VAR_NAME} patterns."""
    def replacer(match):
        return os.environ.get(match.group(1), "")
    return re.sub(r"\$\{env:([^}]+)\}", replacer, value or "")


def _build_env(transport: dict) -> dict:
    base = dict(os.environ)
    for k, v in (transport.get("env") or {}).items():
        base[k] = _resolve_env(v)
    return base


def _spawn_mcp(transport: dict):
    """Spawn an MCP stdio server process."""
    import platform
    command = transport.get("command", "")
    args    = [_resolve_env(a) for a in transport.get("args", [])]
    env     = _build_env(transport)

    # On Windows, shell commands like npx need shell=True or the .cmd extension
    if platform.system() == "Windows":
        return subprocess.Popen(
            [command] + args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=env,
            text=True,
            shell=True,
        )

    return subprocess.Popen(
        [command] + args,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
        text=True,
    )


def _communicate(proc, *messages: dict, timeout: int = 30):
    """Send JSON-RPC messages and return all response lines."""
    payload = "".join(json.dumps(m) + "\n" for m in messages)
    stdout, stderr = proc.communicate(input=payload, timeout=timeout)
    lines = []
    for line in stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            lines.append(json.loads(line))
        except json.JSONDecodeError:
            pass
    return lines, stderr


# ── Public API ────────────────────────────────────────────────────────────────

def list_stdio_tools(transport: dict) -> list[dict]:
    """
    Connect to a stdio MCP server and return its tools/list result
    as a list of OpenAI-compatible tool schemas.
    """
    try:
        proc = _spawn_mcp(transport)
        lines, _ = _communicate(proc, 
            {"jsonrpc": "2.0", "id": 0, "method": "initialize",
             "params": {"protocolVersion": "2024-11-05", "capabilities": {},
                        "clientInfo": {"name": "af-runtime", "version": "0.1.0"}}},
            {"jsonrpc": "2.0", "method": "notifications/initialized"},
            {"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}},
        )

        for msg in lines:
            if msg.get("id") == 1 and "result" in msg:
                return msg["result"].get("tools", [])
        return []
    except Exception:
        return []


def execute_http_tool(url: str, method: str = "GET") -> dict:
    """Execute an HTTP transport tool call."""
    try:
        response = requests.request(method, url)
        return {
            "success": True,
            "status_code": response.status_code,
            "data": response.text,
        }
    except Exception as e:
        return {"success": False, "error": str(e)}


def execute_stdio_tool(transport: dict, tool_name: str, arguments: dict) -> dict:
    """
    Spawn a stdio MCP server, run initialize handshake,
    then call tools/call and return the result.
    """
    try:
        proc = _spawn_mcp(transport)
        lines, stderr = _communicate(proc,
            {"jsonrpc": "2.0", "id": 0, "method": "initialize",
             "params": {"protocolVersion": "2024-11-05", "capabilities": {},
                        "clientInfo": {"name": "af-runtime", "version": "0.1.0"}}},
            {"jsonrpc": "2.0", "method": "notifications/initialized"},
            {"jsonrpc": "2.0", "id": 1, "method": "tools/call",
             "params": {"name": tool_name, "arguments": arguments}},
        )

        for msg in lines:
            if msg.get("id") == 1:
                if "error" in msg:
                    return {"success": False, "error": str(msg["error"])}
                content = msg.get("result", {}).get("content", [])
                text = "\n".join(
                    c.get("text", "") for c in content if c.get("type") == "text"
                )
                return {"success": True, "data": text}

        return {
            "success": False,
            "error": f"No response from MCP server. stderr: {stderr[:300]}",
        }

    except subprocess.TimeoutExpired:
        proc.kill()
        return {"success": False, "error": "MCP server timed out"}
    except Exception as e:
        return {"success": False, "error": str(e)}


# Legacy entry point — kept for backward compatibility
def execute_tool_call(url: str, method: str = "GET") -> dict:
    return execute_http_tool(url, method)
