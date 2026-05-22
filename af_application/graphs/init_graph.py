from langgraph.graph import StateGraph, END

from schemas.state import AgentState

from nodes.clarify import clarify_node
from nodes.generate import generate_node
from nodes.validate import validate_node
from nodes.export import export_node


builder = StateGraph(AgentState)


# Register nodes
builder.add_node("clarify", clarify_node)
builder.add_node("generate", generate_node)
builder.add_node("validate", validate_node)
builder.add_node("export", export_node)


# Define workflow
builder.set_entry_point("clarify")

builder.add_edge("clarify", "generate")
builder.add_edge("generate", "validate")
builder.add_edge("validate", "export")
builder.add_edge("export", END)


# Compile graph
graph = builder.compile()