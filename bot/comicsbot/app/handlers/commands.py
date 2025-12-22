import logging
import time
import httpx

from aiogram import Router
from aiogram.filters import CommandStart, Command, CommandObject
from aiogram.types import Message
from aiogram.fsm.context import FSMContext

from app.keyboards.inline import browse_kb, search_kb, random_kb, mycomics_kb
from app.states import BrowseState
from app.settings import SEARCH_LIMIT_DEFAULT
from app.utils.comics import center_text, comic_text_fallback
from app.services.msg_ctx import put_ctx
from app.services.session import (
    get_or_login_token,
    ensure_fav_ids_map,
    is_saved,
)

router = Router()
log = logging.getLogger(__name__)


async def show_comic(message: Message, comic, pos: int, total: int, saved: bool) -> Message:
    kb = browse_kb(pos > 1, pos < total, center_text(comic.id, pos, total), saved=saved)

    if getattr(comic, "url", ""):
        return await message.answer_photo(
            photo=comic.url,
            caption=f"🖼️ xkcd #{comic.id}",
            reply_markup=kb,
        )

    return await message.answer(comic_text_fallback(comic.id), reply_markup=kb)


@router.message(CommandStart())
async def cmd_start(message: Message):
    await message.answer(
        f"👋 Привет, {message.from_user.first_name}!\n"
        "Это бот с комиксами xkcd.\n\n"
        "/comics - начать просмотр\n"
        "/comics <id> - открыть по id\n"
        "/random - случайный\n"
        "/search <фраза> - поиск\n"
        "/mycomics - мои избранные\n"
        "/help - помощь\n"
    )


@router.message(Command("help"))
async def cmd_help(message: Message):
    await message.answer(
        "ℹ️ Команды:\n\n"
        "/start - приветствие\n"
        "/comics - листать комиксы\n"
        "/comics <id> - открыть комикс по id\n"
        "/random - случайный комикс\n"
        "/search <фраза> - поиск\n"
        "/mycomics - избранное\n"
    )


@router.message(Command("random"))
async def cmd_random(message: Message, state: FSMContext):
    api = (await state.get_data())["api"]

    await ensure_fav_ids_map(state, api, message.from_user, force=False)

    try:
        comic = await api.random_comic()
    except httpx.HTTPError:
        await message.answer("❌ Не могу получить случайный комикс. Сервер недоступен.")
        return

    await state.set_state(BrowseState.browsing)

    saved = await is_saved(state, api, message.from_user, comic.id)
    kb = random_kb(comic.id, saved=saved)

    if comic.url:
        msg = await message.answer_photo(
            photo=comic.url,
            caption=f"🎲 Случайный xkcd #{comic.id}",
            reply_markup=kb,
        )
    else:
        msg = await message.answer(
            comic_text_fallback(comic.id, title="🎲 Случайный xkcd"),
            reply_markup=kb,
        )

    await put_ctx(state, msg.message_id, {"mode": "random", "comic_id": comic.id})


@router.message(Command("mycomics"))
async def cmd_mycomics(message: Message, state: FSMContext):
    api = (await state.get_data())["api"]

    try:
        token = await get_or_login_token(state, api, message.from_user)
    except httpx.HTTPError:
        await message.answer("❌ Auth недоступен. Попробуй позже.")
        return

    try:
        favs = await api.favorites_list(token=token)
    except httpx.HTTPError:
        await message.answer("❌ Favorites недоступен. Попробуй позже.")
        return

    if not favs.items:
        await message.answer("⭐️ Избранное пустое. Нажми «⭐️ Сохранить» на комиксе 🙂")
        return

    # обновим кэш избранного
    ids_map = {str(int(it.comic_id)): True for it in favs.items}
    data = await state.get_data()
    fav_cache = data.get("fav_cache") or {}
    fav_cache[str(message.from_user.id)] = {"ts": int(time.time()), "ids": ids_map}
    await state.update_data(fav_cache=fav_cache)

    ids = [int(it.comic_id) for it in favs.items if int(it.comic_id) > 0]
    total = len(ids)

    idx = 0
    comic_id = ids[idx]

    try:
        comic = await api.comic_by_id(comic_id)
    except httpx.HTTPError:
        await message.answer("❌ Не могу открыть комикс из избранного 😔")
        return

    await state.set_state(BrowseState.browsing)

    center = center_text(comic_id, idx + 1, total)
    kb = mycomics_kb(can_prev=False, can_next=(total > 1), center_text=center)

    if comic.url:
        msg = await message.answer_photo(
            photo=comic.url,
            caption=f"⭐️ Избранное · xkcd #{comic.id}",
            reply_markup=kb,
        )
    else:
        msg = await message.answer(
            comic_text_fallback(comic.id, title="⭐️ Избранное · xkcd"),
            reply_markup=kb,
        )

    await put_ctx(state, msg.message_id, {"mode": "mycomics", "idx": idx, "ids": ids, "comic_id": comic_id})
    log.info("mycomics_open", extra={"tg_id": message.from_user.id, "count": total})


@router.message(Command("comics"))
async def cmd_comics(message: Message, state: FSMContext, command: CommandObject):
    api = (await state.get_data())["api"]
    arg = (command.args or "").strip()

    await ensure_fav_ids_map(state, api, message.from_user, force=False)
    await state.set_state(BrowseState.browsing)

    if arg:
        try:
            comic_id = int(arg)
        except ValueError:
            await message.answer("❌ Пример: /comics 22")
            return

        try:
            comic = await api.comic_by_id(comic_id)
        except httpx.HTTPStatusError as e:
            if e.response.status_code == 404:
                await message.answer("❌ Комикс с таким id не найден.")
                return
            await message.answer("❌ Ошибка при получении комикса. Попробуй позже.")
            return
        except httpx.HTTPError:
            await message.answer("❌ Не могу связаться с сервером комиксов.")
            return

        try:
            first_page = await api.comics_page(page=1, limit=1)
            total = first_page.total
        except httpx.HTTPError:
            total = comic_id

        saved = await is_saved(state, api, message.from_user, comic.id)
        msg = await show_comic(message, comic, pos=comic_id, total=total, saved=saved)

        await put_ctx(state, msg.message_id, {"mode": "by_id", "comic_id": comic_id, "total": total})
        return

    try:
        page = 1
        res = await api.comics_page(page=page, limit=1)
    except httpx.HTTPError:
        await message.answer("❌ Не могу получить список комиксов. Сервер недоступен.")
        return

    if not res.comics:
        await message.answer("Пока нет комиксов.")
        return

    comic = res.comics[0]
    saved = await is_saved(state, api, message.from_user, comic.id)
    msg = await show_comic(message, comic, pos=page, total=res.total, saved=saved)

    await put_ctx(state, msg.message_id, {"mode": "all", "page": page, "total": res.total, "limit": 1, "comic_id": comic.id})


@router.message(Command("search"))
async def cmd_search(message: Message, state: FSMContext, command: CommandObject):
    api = (await state.get_data())["api"]
    phrase = (command.args or "").strip()
    if not phrase:
        await message.answer("❌ Пример: /search linux cpu video")
        return

    await ensure_fav_ids_map(state, api, message.from_user, force=False)

    try:
        res = await api.search(phrase=phrase, limit=SEARCH_LIMIT_DEFAULT)
    except httpx.HTTPStatusError:
        await message.answer("Ничего не нашёл 😔")
        return
    except httpx.HTTPError:
        await message.answer("❌ Поиск временно недоступен. Попробуй позже.")
        return

    if not res.comics:
        await message.answer("Ничего не нашёл 😔")
        return

    total_found = int(res.total)
    results = [c.model_dump() for c in res.comics]
    shown = len(results)

    await message.answer(f"🔎 Нашёл: {total_found}\nЗапрос: {phrase}")

    await state.set_state(BrowseState.browsing)

    idx = 0
    first = results[idx]
    shown_total = shown

    comic_id = int(first["id"])
    saved = await is_saved(state, api, message.from_user, comic_id)

    can_prev = False
    can_next = shown_total > 1
    center = center_text(comic_id, idx + 1, shown_total)

    if first.get("url"):
        msg = await message.answer_photo(
            photo=first["url"],
            caption=f"🔎 xkcd #{comic_id}",
            reply_markup=search_kb(can_prev, can_next, center, saved=saved),
        )
    else:
        msg = await message.answer(
            comic_text_fallback(comic_id, title="🔎 xkcd"),
            reply_markup=search_kb(can_prev, can_next, center, saved=saved),
        )

    await put_ctx(state, msg.message_id, {"mode": "search", "idx": idx, "results": results, "total_shown": shown_total, "comic_id": comic_id})
