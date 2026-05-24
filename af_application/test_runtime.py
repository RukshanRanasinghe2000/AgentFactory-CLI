"""
Runtime smoke test — loads a real AFM spec and runs a single chat turn.

Run from af_application/:
    python test_runtime.py
    python test_runtime.py test_data/weather.md "What is the weather in London?"
"""

import sys
import json
from adapters.spec_loader import load_spec
from graphs.runtime_graph import runtime_graph


# ── Defaults ──────────────────────────────────────────────────────────────────
DEFAULT_SPEC  = "test_data/weather.md"
DEFAULT_INPUT = "What is the weather forecast for Colombo?"


def run(spec_path: str, user_input: str) -> None:
    print("\n" + "─" * 60)
    print(f"  Spec   : {spec_path}")
    print(f"  Input  : {user_input}")
    print("─" * 60 + "\n")

    # Load and validate the spec
    try:
        agent_spec = load_spec(spec_path)
    except Exception as e:
        print(f"[ERROR] Failed to load spec: {e}")
        sys.exit(1)

    print(f"  Agent  : {agent_spec.get('name')}")
    print(f"  Tools  : {[t['name'] for t in agent_spec.get('tools', {}).get('mcp', [])]}\n")

    # Build initial runtime state
    initial_state = {
        "agent_spec":          agent_spec,
        "user_input":          user_input,
        "messages":            [],
        "assistant_response":  None,
        "tool_calls":          [],
        "tool_results":        [],
        "errors":              [],
        "finished":            True,   # single-turn: stop after one cycle
    }

    # Run the graph
    result = runtime_graph.invoke(initial_state)

    # ── Output ────────────────────────────────────────────────────────────────
    print("─" * 60)

    if result.get("tool_calls"):
        print("  Tool calls requested:")
        for tc in result["tool_calls"]:
            print(f"    • {tc['tool']}({json.dumps(tc['arguments'])})")
        print()

    if result.get("tool_results"):
        print("  Tool results:")
        for tr in result["tool_results"]:
            status = "✓" if tr.get("success") else "✗"
            print(f"    {status} {tr['tool']}")
        print()

    print("  Assistant response:")
    print()
    response = result.get("assistant_response") or "(no response)"
    for line in response.splitlines():
        print(f"    {line}")

    print("\n" + "─" * 60 + "\n")


if __name__ == "__main__":
    spec_path  = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_SPEC
    user_input = sys.argv[2] if len(sys.argv) > 2 else DEFAULT_INPUT
    run(spec_path, user_input)
