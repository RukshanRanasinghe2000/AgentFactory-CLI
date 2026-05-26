import json
import os
from typing import Any
from openai import OpenAI, BadRequestError
from dotenv import load_dotenv

load_dotenv()

client = OpenAI(
    api_key=os.environ["MODEL_API_KEY"],
    base_url=os.environ.get("MODEL_BASE_URL"),
)

MODEL_NAME = os.environ.get("MODEL_NAME", "gpt-4o-mini")

# Groq struggles with large tool lists — cap at this many
MAX_TOOLS = 8

# Core filesystem tools to keep when trimming (most useful for code exploration)
_PRIORITY_TOOLS = {
    "list_directory", "directory_tree", "read_text_file",
    "read_multiple_files", "search_files", "sequentialthinking",
    "get_weather", "weather_api",
}


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
    Handles Groq tool_use_failed by retrying without tools.
    """
    full_messages = []
    if system_prompt:
        full_messages.append({"role": "system", "content": system_prompt})
    full_messages.extend(messages)

    # Trim tool list to avoid Groq hallucinating malformed calls
    openai_tools = _trim_tools(tools) if tools else None

    kwargs: dict[str, Any] = {"model": MODEL_NAME, "messages": full_messages}
    if openai_tools:
        kwargs["tools"] = openai_tools
        kwargs["tool_choice"] = "auto"

    try:
        response = client.chat.completions.create(**kwargs)
    except BadRequestError as e:
        # Groq tool_use_failed — retry without tools so user gets a response
        if "tool_use_failed" in str(e) or "tool" in str(e).lower():
            kwargs.pop("tools", None)
            kwargs.pop("tool_choice", None)
            response = client.chat.completions.create(**kwargs)
        else:
            raise

    choice  = response.choices[0]
    message = choice.message

    parsed_tool_calls = []
    if message.tool_calls:
        for tc in message.tool_calls:
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
        "response":      message.content or "",
        "tool_calls":    parsed_tool_calls,
        "finish_reason": choice.finish_reason,
    }


def _trim_tools(tools: list[dict]) -> list[dict]:
    """
    Keep only the most useful tools when the list exceeds MAX_TOOLS.
    Priority tools are kept first; the rest are trimmed.
    """
    if len(tools) <= MAX_TOOLS:
        return tools

    priority = [t for t in tools if t.get("function", {}).get("name") in _PRIORITY_TOOLS]
    others   = [t for t in tools if t.get("function", {}).get("name") not in _PRIORITY_TOOLS]

    trimmed = (priority + others)[:MAX_TOOLS]
    return trimmed
