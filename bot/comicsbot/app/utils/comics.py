def center_text(comic_id: int, pos: int, total: int) -> str:
    return f"#{comic_id}  {pos}/{total}"


def comic_text_fallback(comic_id: int, title: str = "🖼️ xkcd") -> str:
    return (
        f"{title} #{comic_id}\n\n"
        "🤷‍♂️ Этот комикс отсутствует.\n"
        "Похоже, его съели хакеры.\n\n"
        "⬅️ ➡️ — можно попробовать соседние 😉"
    )
