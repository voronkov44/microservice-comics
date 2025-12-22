from pydantic import BaseModel


class ComicRef(BaseModel):
    id: int
    url: str


class ComicsPage(BaseModel):
    comics: list[ComicRef]
    total: int


class FavoriteItem(BaseModel):
    comic_id: int
    created_at_unix: int


class FavoritesList(BaseModel):
    items: list[FavoriteItem]
