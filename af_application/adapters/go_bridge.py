import json
import sys
import os

# Ensure af_application root is on the path regardless of cwd
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from graphs.clarify_graph import clarify_graph
from graphs.generate_graph import generate_graph


def run():
    """
    Two-phase bridge for the Go CLI.

    Phase "clarify"  → runs clarify node, returns questions to Go.
    Phase "generate" → runs generate → validate → export, returns final spec.

    Input JSON must include a "phase" field: "clarify" | "generate"
    """
    try:
        input_data = json.load(sys.stdin)
        phase = input_data.pop("phase", "generate")

        if phase == "clarify":
            result = clarify_graph.invoke(input_data)
        else:
            result = generate_graph.invoke(input_data)

        # Serialize — convert any non-serializable values to str
        print(json.dumps(result, indent=2, default=str))

    except Exception as e:
        print(json.dumps({"success": False, "error": str(e)}))


if __name__ == "__main__":
    run()
