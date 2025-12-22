from aiogram import Router

from app.handlers.commands import router as commands_router
from app.handlers.callbacks import router as callbacks_router

router = Router()
router.include_router(commands_router)
router.include_router(callbacks_router)
