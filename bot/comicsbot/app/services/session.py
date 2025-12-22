import time
import httpx
from aiogram.fsm.context import FSMContext
from app.settings import FAV_CACHE_TTL_SEC


async def get_or_login_token(state: FSMContext, api, tg_user) -> str:
    data = await state.get_data()
    tokens = data.get("tokens") or {}
    key = str(tg_user.id)

    if tokens.get(key):
        return tokens[key]

    token = await api.bot_login_telegram(
        tg_id=tg_user.id,
        username=tg_user.username or "",
        first_name=tg_user.first_name or "",
        last_name=tg_user.last_name or "",
    )

    tokens[key] = token
    await state.update_data(tokens=tokens)
    return token


async def ensure_fav_ids_map(state: FSMContext, api, tg_user, force: bool = False) -> dict:
    """
    Возвращает dict {"<comic_id>": True, ...} из кэша FSM
    При необходимости обновляет через /api/mycomics
    """
    data = await state.get_data()
    fav_cache = data.get("fav_cache") or {}
    key = str(tg_user.id)
    entry = fav_cache.get(key) or {}
    now = int(time.time())

    if not force and entry.get("ids") and (now - int(entry.get("ts", 0)) < FAV_CACHE_TTL_SEC):
        return entry["ids"]

    try:
        token = await get_or_login_token(state, api, tg_user)
        favs = await api.favorites_list(token=token)
        ids = {str(int(it.comic_id)): True for it in favs.items}
        fav_cache[key] = {"ts": now, "ids": ids}
        await state.update_data(fav_cache=fav_cache)
        return ids
    except httpx.HTTPError:
        # если сеть упала - вернём то, что было
        if entry.get("ids"):
            return entry["ids"]
        return {}


async def is_saved(state: FSMContext, api, tg_user, comic_id: int) -> bool:
    ids = await ensure_fav_ids_map(state, api, tg_user, force=False)
    return bool(ids.get(str(int(comic_id))))


async def set_saved_in_cache(state: FSMContext, tg_user, comic_id: int, saved: bool) -> None:
    data = await state.get_data()
    fav_cache = data.get("fav_cache") or {}
    key = str(tg_user.id)
    entry = fav_cache.get(key) or {"ts": int(time.time()), "ids": {}}
    ids = entry.get("ids") or {}

    cid = str(int(comic_id))
    if saved:
        ids[cid] = True
    else:
        ids.pop(cid, None)

    entry["ids"] = ids
    entry["ts"] = int(time.time())
    fav_cache[key] = entry
    await state.update_data(fav_cache=fav_cache)
