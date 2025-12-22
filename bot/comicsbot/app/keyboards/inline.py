from aiogram.types import InlineKeyboardMarkup, InlineKeyboardButton


def _fav_button(saved: bool) -> InlineKeyboardButton:
    if saved:
        return InlineKeyboardButton(text="✅ Сохранено", callback_data="fav:noop")
    return InlineKeyboardButton(text="⭐️ Сохранить", callback_data="fav:save")


def browse_kb(can_prev: bool, can_next: bool, center_text: str, saved: bool = False) -> InlineKeyboardMarkup:
    row1 = []
    if can_prev:
        row1.append(InlineKeyboardButton(text="⬅️", callback_data="nav:prev"))

    row1.append(InlineKeyboardButton(text=center_text, callback_data="nav:noop"))

    if can_next:
        row1.append(InlineKeyboardButton(text="➡️", callback_data="nav:next"))

    row2 = [_fav_button(saved)]
    return InlineKeyboardMarkup(inline_keyboard=[row1, row2])


def search_kb(can_prev: bool, can_next: bool, center_text: str, saved: bool = False) -> InlineKeyboardMarkup:
    row1 = []
    if can_prev:
        row1.append(InlineKeyboardButton(text="⬅️", callback_data="nav:prev"))

    row1.append(InlineKeyboardButton(text=center_text, callback_data="nav:noop"))

    if can_next:
        row1.append(InlineKeyboardButton(text="➡️", callback_data="nav:next"))

    rows = [row1, [_fav_button(saved)]]
    return InlineKeyboardMarkup(inline_keyboard=rows)


def random_kb(comic_id: int, saved: bool = False) -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [InlineKeyboardButton(text="🎲 Ещё", callback_data="rnd:next")],
            [_fav_button(saved)],
        ]
    )


def mycomics_kb(can_prev: bool, can_next: bool, center_text: str) -> InlineKeyboardMarkup:
    row1 = []
    if can_prev:
        row1.append(InlineKeyboardButton(text="⬅️", callback_data="nav:prev"))

    row1.append(InlineKeyboardButton(text=center_text, callback_data="nav:noop"))

    if can_next:
        row1.append(InlineKeyboardButton(text="➡️", callback_data="nav:next"))

    row2 = [InlineKeyboardButton(text="🗑️ Удалить", callback_data="fav:del")]
    return InlineKeyboardMarkup(inline_keyboard=[row1, row2])
