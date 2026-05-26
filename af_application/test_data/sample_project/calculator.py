class Calculator:
    """Evaluates simple arithmetic expressions."""

    def evaluate(self, expression: str) -> float | None:
        """
        Parse and evaluate an arithmetic expression string.
        Supports: +, -, *, /
        Returns None on error.
        """
        try:
            # Only allow safe characters
            allowed = set("0123456789+-*/(). ")
            if not all(c in allowed for c in expression):
                print("Error: unsupported characters in expression")
                return None

            result = eval(expression)  # safe — restricted to numbers and operators
            return result

        except ZeroDivisionError:
            print("Error: division by zero")
            return None
        except Exception as e:
            print(f"Error: {e}")
            return None
