from langgraph.graph import StateGraph, END

from schemas.runtime_state import RuntimeState

from nodes.chat import chat_node
from nodes.router import router_node
from nodes.tools import tools_node
from nodes.response import response_node


builder = StateGraph(RuntimeState)


# Register nodes
builder.add_node("chat", chat_node)
builder.add_node("router", router_node)
builder.add_node("tools", tools_node)
builder.add_node("response", response_node)


# Entry point
builder.set_entry_point("chat")


# Workflow edges
builder.add_edge("chat", "router")


# Conditional routing
def route_tools(state: RuntimeState):
    tool_calls = state.get("tool_calls") or []
    if tool_calls:
        return "tools"
    return "response"


builder.add_conditional_edges(
    "router",
    route_tools,
    {
        "tools": "tools",
        "response": "response",
    },
)


# After tools execute,
# generate final response
builder.add_edge("tools", "response")


# Runtime loop
def should_continue(state: RuntimeState):
    """
    Continue chat loop unless session finished.
    """

    if state.get("finished"):
        return END

    return "chat"


builder.add_conditional_edges(
    "response",
    should_continue,
    {
        "chat": "chat",
        END: END,
    },
)


# Compile graph
runtime_graph = builder.compile()