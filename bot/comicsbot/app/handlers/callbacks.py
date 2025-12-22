import logging

import httpx
from aiogram import Router, F
from aiogram.fsm.context import FSMContext
from aiogram.types import CallbackQuery

from app.keyboards.inline import browse_kb, search_kb, random_kb, mycomics_kb
from app.services.msg_ctx import get_ctx, put_ctx, drop_ctx
from app.services.session import (
    get_or_login_token,
    is_saved,
    set_saved_in_cache,
)
from app.services.tg_edit import edit_or_replace_comic
from app.utils.comics import center_text, comic_text_fallback

router = Router()
log = logging.getLogger(__name__)


def _caption_for_mode(mode: str, comic_id: int) -> str:
    if mode == "random":
        return f"🎲 Случайный xkcd #{comic_id}"
    if mode == "search":
        return f"🔎 xkcd #{comic_id}"
    if mode == "mycomics":
        return f"⭐️ Избранное · xkcd #{comic_id}"
    return f"🖼️ xkcd #{comic_id}"


def _fallback_for_mode(mode: str, comic_id: int) -> str:
    if mode == "random":
        return comic_text_fallback(comic_id, title="🎲 Случайный xkcd")
    if mode == "search":
        return comic_text_fallback(comic_id, title="🔎 xkcd")
    if mode == "mycomics":
        return comic_text_fallback(comic_id, title="⭐️ Избранное · xkcd")
    return comic_text_fallback(comic_id, title="🖼️ xkcd")


def _current_comic_id(ctx: dict) -> int | None:
    if ctx.get("comic_id"):
        try:
            return int(ctx["comic_id"])
        except (TypeError, ValueError):
            return None

    mode = ctx.get("mode")

    if mode == "search":
        results = ctx.get("results") or []
        idx = int(ctx.get("idx", 0) or 0)
        if 0 <= idx < len(results):
            return int(results[idx]["id"])

    if mode == "mycomics":
        ids = ctx.get("ids") or []
        idx = int(ctx.get("idx", 0) or 0)
        if 0 <= idx < len(ids):
            return int(ids[idx])

    return None


async def _replace_and_rebind_ctx(
    call: CallbackQuery,
    state: FSMContext,
    old_msg_id: int,
    new_ctx: dict,
    *,
    url: str,
    caption: str,
    text: str,
    reply_markup,
) -> None:
    new_msg = await edit_or_replace_comic(
        call,
        url=url,
        caption=caption,
        text=text,
        reply_markup=reply_markup,
    )

    if new_msg.message_id != old_msg_id:
        await drop_ctx(state, old_msg_id)

    await put_ctx(state, new_msg.message_id, new_ctx)


@router.callback_query(F.data == "fav:noop")
async def on_fav_noop(call: CallbackQuery, state: FSMContext):
    await call.answer("⭐️ Уже в избранном", show_alert=False)


@router.callback_query(F.data.startswith("nav:"))
async def on_nav(call: CallbackQuery, state: FSMContext):
    await call.answer()

    if not call.message:
        return

    action = call.data.split(":", 1)[1]
    if action == "noop":
        return

    data = await state.get_data()
    api = data["api"]

    old_msg_id = call.message.message_id
    ctx = await get_ctx(state, old_msg_id)
    if not ctx:
        await call.answer("Контекст устарел. Запусти команду заново 🙂", show_alert=False)
        return

    mode = ctx.get("mode")

    # all
    if mode == "all":
        page = int(ctx.get("page", 1))
        total = int(ctx.get("total", 1))
        limit = int(ctx.get("limit", 1))

        if action == "prev" and page > 1:
            page -= 1
        elif action == "next" and page < total:
            page += 1
        else:
            return

        res = await api.comics_page(page=page, limit=limit)
        if not res.comics:
            return

        comic = res.comics[0]
        ctx.update({"page": page, "total": int(res.total), "comic_id": int(comic.id)})

        saved = await is_saved(state, api, call.from_user, comic.id)
        kb = browse_kb(
            can_prev=page > 1,
            can_next=page < int(res.total),
            center_text=center_text(comic.id, page, int(res.total)),
            saved=saved,
        )

        await _replace_and_rebind_ctx(
            call, state, old_msg_id, ctx,
            url=comic.url or "",
            caption=_caption_for_mode("all", comic.id),
            text=_fallback_for_mode("all", comic.id),
            reply_markup=kb,
        )
        return

    # by_id
    if mode == "by_id":
        comic_id = int(ctx.get("comic_id", 1))
        total = int(ctx.get("total", 1))

        if action == "prev" and comic_id > 1:
            comic_id -= 1
        elif action == "next" and comic_id < total:
            comic_id += 1
        else:
            return

        comic = await api.comic_by_id(comic_id)
        ctx["comic_id"] = comic_id

        saved = await is_saved(state, api, call.from_user, comic_id)
        kb = browse_kb(
            can_prev=comic_id > 1,
            can_next=comic_id < total,
            center_text=center_text(comic_id, comic_id, total),
            saved=saved,
        )

        await _replace_and_rebind_ctx(
            call, state, old_msg_id, ctx,
            url=comic.url or "",
            caption=_caption_for_mode("by_id", comic_id),
            text=_fallback_for_mode("by_id", comic_id),
            reply_markup=kb,
        )
        return

    # search
    if mode == "search":
        results = ctx.get("results") or []
        total_shown = int(ctx.get("total_shown", len(results)))
        idx = int(ctx.get("idx", 0))

        if not results:
            return

        if action == "prev":
            if idx <= 0:
                return
            idx -= 1
        elif action == "next":
            if idx >= total_shown - 1:
                return
            idx += 1
        else:
            return

        c = results[idx]
        comic_id = int(c["id"])
        ctx.update({"idx": idx, "comic_id": comic_id})

        saved = await is_saved(state, api, call.from_user, comic_id)
        kb = search_kb(
            can_prev=idx > 0,
            can_next=idx < total_shown - 1,
            center_text=center_text(comic_id, idx + 1, total_shown),
            saved=saved,
        )

        await _replace_and_rebind_ctx(
            call, state, old_msg_id, ctx,
            url=(c.get("url") or ""),
            caption=_caption_for_mode("search", comic_id),
            text=_fallback_for_mode("search", comic_id),
            reply_markup=kb,
        )
        return

    # mycomics
    if mode == "mycomics":
        ids = ctx.get("ids") or []
        if not ids:
            return

        idx = int(ctx.get("idx", 0))
        total = len(ids)

        if action == "prev":
            if idx <= 0:
                return
            idx -= 1
        elif action == "next":
            if idx >= total - 1:
                return
            idx += 1
        else:
            return

        comic_id = int(ids[idx])
        comic = await api.comic_by_id(comic_id)

        ctx.update({"idx": idx, "comic_id": comic_id})

        kb = mycomics_kb(
            can_prev=idx > 0,
            can_next=idx < total - 1,
            center_text=center_text(comic_id, idx + 1, total),
        )

        await _replace_and_rebind_ctx(
            call, state, old_msg_id, ctx,
            url=comic.url or "",
            caption=_caption_for_mode("mycomics", comic_id),
            text=_fallback_for_mode("mycomics", comic_id),
            reply_markup=kb,
        )
        return


@router.callback_query(F.data == "fav:save")
async def on_fav_save(call: CallbackQuery, state: FSMContext):
    if not call.message:
        return

    old_msg_id = call.message.message_id
    ctx = await get_ctx(state, old_msg_id)
    if not ctx:
        await call.answer("Контекст устарел. Запусти команду заново 🙂", show_alert=False)
        return

    comic_id = _current_comic_id(ctx)
    if not comic_id or comic_id <= 0:
        await call.answer("Не понял, какой комикс сохранять 😔", show_alert=False)
        return

    data = await state.get_data()
    api = data["api"]
    tg = call.from_user

    try:
        token = await get_or_login_token(state, api, tg)
        code = await api.favorites_add(token=token, comic_id=comic_id)
    except httpx.HTTPError:
        await call.answer("Auth/Favorites недоступен 😔", show_alert=False)
        return

    if code in (200, 204, 409):
        await set_saved_in_cache(state, tg, comic_id, saved=True)
        # клава перерисовывается без перезапроса
        mode = ctx.get("mode")
        if mode == "all":
            page = int(ctx.get("page", 1))
            total = int(ctx.get("total", 1))
            await call.message.edit_reply_markup(
                reply_markup=browse_kb(page > 1, page < total, center_text(comic_id, page, total), saved=True)
            )
        elif mode == "by_id":
            total = int(ctx.get("total", 1))
            await call.message.edit_reply_markup(
                reply_markup=browse_kb(comic_id > 1, comic_id < total, center_text(comic_id, comic_id, total), saved=True)
            )
        elif mode == "search":
            idx = int(ctx.get("idx", 0))
            total_shown = int(ctx.get("total_shown", 1))
            await call.message.edit_reply_markup(
                reply_markup=search_kb(idx > 0, idx < total_shown - 1, center_text(comic_id, idx + 1, total_shown), saved=True)
            )
        elif mode == "random":
            await call.message.edit_reply_markup(reply_markup=random_kb(comic_id, saved=True))

        await call.answer("⭐️ Сохранил!", show_alert=False)
        return

    if code == 404:
        await call.answer("Комикс не найден 😔", show_alert=False)
        return
    if code == 401:
        await call.answer("Не смог авторизоваться (401) 😔", show_alert=False)
        return

    await call.answer(f"Не смог сохранить ({code}) 😔", show_alert=False)


@router.callback_query(F.data == "fav:del")
async def on_fav_del(call: CallbackQuery, state: FSMContext):
    if not call.message:
        return

    old_msg_id = call.message.message_id
    ctx = await get_ctx(state, old_msg_id)
    if not ctx:
        await call.answer("Контекст устарел. Запусти /mycomics заново 🙂", show_alert=False)
        return

    if ctx.get("mode") != "mycomics":
        await call.answer("Удалять удобнее из /mycomics 🙂", show_alert=False)
        return

    comic_id = _current_comic_id(ctx)
    if not comic_id:
        await call.answer("Не понял, что удалять 😔", show_alert=False)
        return

    data = await state.get_data()
    api = data["api"]
    tg = call.from_user

    try:
        token = await get_or_login_token(state, api, tg)
        code = await api.favorites_delete(token=token, comic_id=comic_id)
    except httpx.HTTPError:
        await call.answer("Auth/Favorites недоступен 😔", show_alert=False)
        return

    if code not in (200, 204, 404):
        if code == 401:
            await call.answer("Не смог авторизоваться (401) 😔", show_alert=False)
            return
        await call.answer(f"Не смог удалить ({code}) 😔", show_alert=False)
        return

    await set_saved_in_cache(state, tg, comic_id, saved=False)

    ids = [int(x) for x in (ctx.get("ids") or []) if str(x).isdigit()]
    if comic_id in ids:
        ids.remove(comic_id)

    if not ids:
        await drop_ctx(state, old_msg_id)
        try:
            await call.message.delete()
        except Exception:
            pass
        await call.message.answer("⭐️ Избранное пустое")
        await call.answer("🗑️ Удалил из избранного", show_alert=False)
        return

    idx = int(ctx.get("idx", 0))
    if idx >= len(ids):
        idx = len(ids) - 1
    if idx < 0:
        idx = 0

    new_id = int(ids[idx])

    try:
        comic = await api.comic_by_id(new_id)
    except httpx.HTTPError:
        await call.answer("Не могу открыть комикс 😔", show_alert=False)
        return

    ctx.update({"ids": ids, "idx": idx, "comic_id": new_id})

    kb = mycomics_kb(
        can_prev=idx > 0,
        can_next=idx < len(ids) - 1,
        center_text=center_text(new_id, idx + 1, len(ids)),
    )

    await _replace_and_rebind_ctx(
        call, state, old_msg_id, ctx,
        url=comic.url or "",
        caption=_caption_for_mode("mycomics", new_id),
        text=_fallback_for_mode("mycomics", new_id),
        reply_markup=kb,
    )

    await call.answer("🗑️ Удалил из избранного", show_alert=False)


@router.callback_query(F.data == "rnd:next")
async def on_random_next(call: CallbackQuery, state: FSMContext):
    await call.answer()

    if not call.message:
        return

    data = await state.get_data()
    api = data["api"]

    try:
        comic = await api.random_comic()
    except httpx.HTTPError:
        await call.answer("Не могу получить случайный комикс 😔", show_alert=False)
        return

    # ctx обновим после replace (вдруг поменяется message_id)
    ctx = {"mode": "random", "comic_id": int(comic.id)}

    saved = await is_saved(state, api, call.from_user, comic.id)
    kb = random_kb(comic.id, saved=saved)

    old_msg_id = call.message.message_id
    await _replace_and_rebind_ctx(
        call, state, old_msg_id, ctx,
        url=comic.url or "",
        caption=_caption_for_mode("random", comic.id),
        text=_fallback_for_mode("random", comic.id),
        reply_markup=kb,
    )
