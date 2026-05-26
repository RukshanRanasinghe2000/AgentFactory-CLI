"""
Manual smoke test for the LangGraph workflow.
Run from af_application/:
    python test_workflow.py
"""

import json
from graphs.init_graph import graph


def test_workflow():
    initial_state = {
        "idea": "A customer support agent that handles refund requests",
        "clarification_questions": [],
        "clarification_answers": [],
        "enriched_idea": "",
        "agent_spec": None,
        "validation_errors": [],
        "exported_file": None,
    }

    print("Running workflow...\n")
    result = graph.invoke(initial_state)

    print("=== Clarification Questions ===")
    for q in result.get("clarification_questions", []):
        print(f"  - {q}")

    print("\n=== Validation Errors ===")
    errors = result.get("validation_errors", [])
    if errors:
        for e in errors:
            print(f"  - {e}")
    else:
        print("  None")

    print("\n=== Exported File ===")
    print(f"  {result.get('exported_file', 'Not exported')}")

    print("\n=== Agent Spec (truncated) ===")
    spec = result.get("agent_spec")
    if spec:
        print(json.dumps(spec, indent=2)[:500] + "...")
    else:
        print("  No spec generated")


if __name__ == "__main__":
    test_workflow()
