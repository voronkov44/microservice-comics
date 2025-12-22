from aiogram.fsm.context import FSMContext
from app.settings import MAX_SEARCH_SESSIONS


async def get_ctx(state: FSMContext, msg_id: int) -> dict | None:
    data = await state.get_data()
    msg_ctx = data.get("msg_ctx") or {}
    return msg_ctx.get(str(msg_id))


async def put_ctx(state: FSMContext, msg_id: int, ctx: dict) -> None:
    data = await state.get_data()
    msg_ctx = data.get("msg_ctx") or {}
    msg_ctx[str(msg_id)] = ctx

    # ограничиваем количество сохранённых контекстов
    if len(msg_ctx) > MAX_SEARCH_SESSIONS:
        oldest_key = next(iter(msg_ctx.keys()))
        msg_ctx.pop(oldest_key, None)

    await state.update_data(msg_ctx=msg_ctx)


async def drop_ctx(state: FSMContext, msg_id: int) -> None:
    data = await state.get_data()
    msg_ctx = data.get("msg_ctx") or {}
    msg_ctx.pop(str(msg_id), None)
    await state.update_data(msg_ctx=msg_ctx)
