class History:
    """Tracks calculation history for the current session."""

    def __init__(self):
        self._entries: list[tuple[str, float]] = []

    def add(self, expression: str, result: float) -> None:
        """Record a completed calculation."""
        self._entries.append((expression, result))

    def show(self) -> None:
        """Print all past calculations."""
        if not self._entries:
            print("No history yet.")
            return
        for i, (expr, result) in enumerate(self._entries, 1):
            print(f"  {i}. {expr} = {result}")
