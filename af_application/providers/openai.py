import os
from typing import Any
from openai import OpenAI
from dotenv import load_dotenv

load_dotenv()

client = OpenAI(
    api_key=os.environ["MODEL_API_KEY"],
    base_url=os.environ.get("MODEL_BASE_URL"),
)

MODEL_NAME = os.environ.get("MODEL_NAME", "gpt-4o-mini")


def generate_text(prompt: str) -> str:
    """Simple single-turn text generation (used by init workflow)."""
    response = client.chat.completions.create(
        model=MODEL_NAME,
        messages=[{"role": "user", "content": prompt}]
    )
    return response.choices[0].message.content


def generate_with_tools(
    messages: list[dict],
    tools: list[dict] | None = None,
    system_prompt: str | None = None,
) -> dict[str, Any]:
    """
    Send a conversation to the LLM with optional tool schemas.

    Returns a dict:
    {
        "response": str,           # assistant text (may be empty if tool_calls present)
        "tool_calls": [            # list of requested tool calls (may be empty)
            {
                "id":        str,
                "tool":      str,   # function name
                "arguments": dict
            }
        ],
        "finish_reason": str       # "stop" | "tool_calls" | "length"
    }
    """
    # Build message list — prepend system prompt if provided
    full_messages = []
    if system_prompt:
        full_messages.append({"role": "system", "content": system_prompt})
    full_messages.extend(messages)

    # Build tool schemas in OpenAI function-calling format
    openai_tools = None
    if tools:
        openai_tools = [_to_openai_tool(t) for t in tools]

    kwargs: dict[str, Any] = {
        "model": MODEL_NAME,
        "messages": full_messages,
    }
    if openai_tools:
        kwargs["tools"] = openai_tools
        kwargs["tool_choice"] = "auto"

    response = client.chat.completions.create(**kwargs)
    choice = response.choices[0]
    message = choice.message

    # Parse tool calls
    parsed_tool_calls = []
    if message.tool_calls:
        for tc in message.tool_calls:
            import json
            try:
                args = json.loads(tc.function.arguments)
            except Exception:
                args = {}
            parsed_tool_calls.append({
                "id":        tc.id,
                "tool":      tc.function.name,
                "arguments": args,
            })

    return {
        "response":     message.content or "",
        "tool_calls":   parsed_tool_calls,
        "finish_reason": choice.finish_reason,
    }


def _to_openai_tool(tool_def: dict) -> dict:
    """
    Convert an AFM tool definition into an OpenAI tool schema.

    AFM tool shape:
    {
        "name": "get_weather",
        "description": "...",
        "transport": { "type": "http", "url": "..." },
        "query_params": [
            { "key": "city", "description": "City name", "required": true }
        ]
    }
    """
    params: dict[str, Any] = {"type": "object", "properties": {}, "required": []}

    for qp in tool_def.get("query_params", []):
        key = qp.get("key", "")
        params["properties"][key] = {
            "type": "string",
            "description": qp.get("description", ""),
        }
        if qp.get("required", False):
            params["required"].append(key)

    return {
        "type": "function",
        "function": {
            "name":        tool_def.get("name", ""),
            "description": tool_def.get("description", ""),
            "parameters":  params,
        },
    }
