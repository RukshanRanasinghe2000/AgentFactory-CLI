from calculator import Calculator
from history import History

def main():
    calc = Calculator()
    history = History()

    print("Simple Calculator — type 'quit' to exit, 'history' to see past results\n")

    while True:
        expr = input("> ").strip()

        if expr == "quit":
            break
        if expr == "history":
            history.show()
            continue

        result = calc.evaluate(expr)
        if result is not None:
            print(f"= {result}")
            history.add(expr, result)

if __name__ == "__main__":
    main()
