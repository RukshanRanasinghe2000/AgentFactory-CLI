import requests


def execute_tool_call(
    url: str,
    method: str = "GET"
):
    """
    Simple runtime tool executor.

    Future support:
    - MCP transports
    - auth
    - retries
    - streaming
    """

    try:
        response = requests.request(
            method,
            url
        )

        return {
            "success": True,
            "status_code": response.status_code,
            "data": response.text
        }

    except Exception as e:
        return {
            "success": False,
            "error": str(e)
        }