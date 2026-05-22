import json
import sys

from graphs.init_graph import graph


def run():
    """
    Receive JSON from Go CLI,
    execute LangGraph workflow,
    return JSON response.
    """

    try:
        input_data = json.load(sys.stdin)

        result = graph.invoke(input_data)

        print(json.dumps(result, indent=2))

    except Exception as e:
        error_response = {
            "success": False,
            "error": str(e)
        }

        print(json.dumps(error_response))


if __name__ == "__main__":
    run()