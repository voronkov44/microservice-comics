import httpx
from .models import ComicRef, ComicsPage, FavoritesList


class ApiClient:
    def __init__(self, base_url: str, internal_url: str, timeout: float = 10.0):
        self.base_url = base_url.rstrip("/")
        self.internal_url = internal_url.rstrip("/")
        self.client = httpx.AsyncClient(timeout=timeout)

    async def close(self):
        await self.client.aclose()

    def _user_headers(self, token: str) -> dict:
        return {"Authorization": f"Token {token}"}

    async def comics_page(self, page: int = 1, limit: int = 1) -> ComicsPage:
        r = await self.client.get(
            f"{self.base_url}/api/comics",
            params={"page": page, "limit": limit},
        )
        r.raise_for_status()
        return ComicsPage.model_validate(r.json())

    async def comic_by_id(self, comic_id: int) -> ComicRef:
        r = await self.client.get(f"{self.base_url}/api/comics/{comic_id}")
        r.raise_for_status()
        return ComicRef.model_validate(r.json())

    async def random_comic(self) -> ComicRef:
        r = await self.client.get(f"{self.base_url}/api/comics/random")
        r.raise_for_status()
        return ComicRef.model_validate(r.json())

    async def search(self, phrase: str, limit: int = 10) -> ComicsPage:
        r = await self.client.get(
            f"{self.base_url}/api/search",
            params={"phrase": phrase, "limit": limit},
        )
        r.raise_for_status()
        return ComicsPage.model_validate(r.json())

    # auth
    async def bot_login_telegram(
        self,
        tg_id: int,
        username: str = "",
        first_name: str = "",
        last_name: str = "",
    ) -> str:
        payload = {
            "tg_id": int(tg_id),
            "username": username or "",
            "first_name": first_name or "",
            "last_name": last_name or "",
        }
        r = await self.client.post(
            f"{self.internal_url}/api/auth/bot/telegram/login",
            json=payload,
        )
        r.raise_for_status()
        data = r.json()
        return data["token"]

    # favorites
    async def favorites_list(self, token: str) -> FavoritesList:
        r = await self.client.get(
            f"{self.base_url}/api/mycomics",
            headers=self._user_headers(token),
        )
        r.raise_for_status()
        return FavoritesList.model_validate(r.json())

    async def favorites_add(self, token: str, comic_id: int) -> int:
        r = await self.client.post(
            f"{self.base_url}/api/mycomics/{int(comic_id)}",
            headers=self._user_headers(token),
        )
        return r.status_code

    async def favorites_delete(self, token: str, comic_id: int) -> int:
        r = await self.client.delete(
            f"{self.base_url}/api/mycomics/{int(comic_id)}",
            headers=self._user_headers(token),
        )
        return r.status_code
