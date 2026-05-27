"""
Runtime bridge for the Go CLI.

Phases:
  load    → parse the spec, return agent name + model info
  chat    → run one turn of the runtime graph (consolechat)
  webhook → run one turn triggered by a platform event (platformchat)
"""

import json
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from adapters.spec_loader import load_spec
from graphs.runtime_graph import runtime_graph

# In-process session store: chat_id → message history
# Keyed by platform + chat_id so multiple chats stay isolated
_sessions: dict[str, list] = {}


def _run_graph(spec: dict, spec_path: str, user_input: str, messages: list) -> dict:
    """Shared graph invocation used by both chat and webhook phases."""
    state = {
        "agent_spec":         spec,
        "spec_path":          spec_path,
        "user_input":         user_input,
        "messages":           messages,
        "assistant_response": None,
        "tool_calls":         [],
        "tool_results":       [],
        "errors":             [],
        "finished":           True,
        "_tool_schemas":      [],
    }
    return runtime_graph.invoke(state)


def run():
    try:
        data      = json.load(sys.stdin)
        phase     = data.get("phase", "chat")
        spec_path = data.get("spec_path", "")

        spec = load_spec(spec_path)

        # ── Load phase ────────────────────────────────────────────────────────
        if phase == "load":
            model = spec.get("model") or {}
            print(json.dumps({
                "success":        True,
                "agent_name":     spec.get("name", "Agent"),
                "spec_version":   spec.get("spec_version") or "",
                "model_provider": model.get("provider", "unknown"),
                "model_name":     model.get("name", "unknown"),
                "interfaces":     spec.get("interfaces", []),
            }))
            return

        # ── Chat phase (consolechat) ───────────────────────────────────────────
        if phase == "chat":
            user_input = data.get("user_input", "")
            messages   = data.get("messages", [])
            result     = _run_graph(spec, spec_path, user_input, messages)
            model      = spec.get("model") or {}
            print(json.dumps({
                "success":            True,
                "agent_name":         spec.get("name", "Agent"),
                "model_provider":     model.get("provider", "unknown"),
                "model_name":         model.get("name", "unknown"),
                "assistant_response": result.get("assistant_response", ""),
                "tool_calls":         result.get("tool_calls", []),
                "messages":           result.get("messages", []),
            }, default=str))
            return

        # ── Webhook phase (platformchat) ──────────────────────────────────────
        if phase == "webhook":
            platform   = data.get("platform", "")
            session_id = data.get("session_id", "default")
            user_input = data.get("user_input", "")

            # Load or create session history for this chat
            key      = f"{platform}:{session_id}"
            messages = _sessions.get(key, [])

            result   = _run_graph(spec, spec_path, user_input, messages)

            # Persist updated history
            _sessions[key] = result.get("messages", [])

            print(json.dumps({
                "success":            True,
                "assistant_response": result.get("assistant_response", ""),
            }, default=str))
            return

        print(json.dumps({"success": False, "error": f"Unknown phase: {phase}"}))

    except Exception as e:
        import traceback
        print(json.dumps({"success": False, "error": traceback.format_exc()}))


if __name__ == "__main__":
    run()
