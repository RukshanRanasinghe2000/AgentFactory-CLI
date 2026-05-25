"""
Runtime bridge for the Go CLI.

Phases:
  load  → parse the spec, return agent name + model info
  chat  → run one turn of the runtime graph, return assistant response
"""

import json
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from adapters.spec_loader import load_spec
from graphs.runtime_graph import runtime_graph


def run():
    try:
        data = json.load(sys.stdin)
        phase     = data.get("phase", "chat")
        spec_path = data.get("spec_path", "")

        spec = load_spec(spec_path)

        # ── Load phase: just return agent metadata ────────────────────────────
        if phase == "load":
            model    = spec.get("model") or {}
            print(json.dumps({
                "success":        True,
                "agent_name":     spec.get("name", "Agent"),
                "model_provider": model.get("provider", "unknown"),
                "model_name":     model.get("name", "unknown"),
            }))
            return

        # ── Chat phase: run one turn of the runtime graph ─────────────────────
        user_input = data.get("user_input", "")
        messages   = data.get("messages", [])

        state = {
            "agent_spec":          spec,
            "user_input":          user_input,
            "messages":            messages,
            "assistant_response":  None,
            "tool_calls":          [],
            "tool_results":        [],
            "errors":              [],
            "finished":            True,
            "_tool_schemas":       [],
        }

        result = runtime_graph.invoke(state)

        model = spec.get("model") or {}
        print(json.dumps({
            "success":            True,
            "agent_name":         spec.get("name", "Agent"),
            "model_provider":     model.get("provider", "unknown"),
            "model_name":         model.get("name", "unknown"),
            "assistant_response": result.get("assistant_response", ""),
            "tool_calls":         result.get("tool_calls", []),
            "messages":           result.get("messages", []),
        }, default=str))

    except Exception as e:
        import traceback
        print(json.dumps({"success": False, "error": traceback.format_exc()}))


if __name__ == "__main__":
    run()
