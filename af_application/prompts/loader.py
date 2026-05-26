from pathlib import Path

PROMPT_DIR = Path(__file__).parent

def load_prompt(filename: str) -> str:
    path = PROMPT_DIR / filename

    with open(path, "r", encoding="utf-8") as f:
        return f.read()