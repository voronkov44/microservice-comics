import asyncio
import logging

from aiogram import Bot, Dispatcher
from aiogram.fsm.storage.memory import MemoryStorage

from config import load_config
from app.handlers.router import router
from app.adapters.api.client import ApiClient


async def main():
    logging.basicConfig(level=logging.INFO)

    cfg = load_config()

    bot = Bot(token=cfg.bot_token)
    dp = Dispatcher(storage=MemoryStorage())

    # rest api client
    api_client = ApiClient(
        base_url=cfg.api_base_url,
        internal_url=cfg.api_internal_base_url,
        timeout=cfg.api_timeout,
    )

    # прокидываем api в dispatcher data
    dp["api"] = api_client

    # middleware-lite: гарантируем, что api есть в FSM
    @dp.update.outer_middleware()
    async def inject_api(handler, event, data):
        state = data.get("state")
        if state:
            cur = await state.get_data()
            if "api" not in cur:
                await state.update_data(api=api_client)
        return await handler(event, data)

    dp.include_router(router)

    try:
        await dp.start_polling(bot)
    finally:
        await api_client.close()
        await bot.session.close()


if __name__ == "__main__":
    asyncio.run(main())
