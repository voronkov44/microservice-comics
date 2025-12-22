from dataclasses import dataclass
import os
from dotenv import load_dotenv

load_dotenv()

@dataclass
class Config:
    bot_token: str
    api_base_url: str
    api_internal_base_url: str
    api_timeout: float = 10.0

def load_config() -> Config:
    bot_token = os.getenv("BOT_TOKEN")
    if not bot_token:
        raise RuntimeError("BOT_TOKEN is not set in .env")

    return Config(
        bot_token=bot_token,
        api_base_url=os.getenv("API_BASE_URL", "http://api:8080"),
        api_internal_base_url=os.getenv("API_INTERNAL_BASE_URL", "http://api:8081"),
        api_timeout=float(os.getenv("API_TIMEOUT", "10")),
    )
