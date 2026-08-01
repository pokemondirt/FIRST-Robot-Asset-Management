from pathlib import Path

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    app_name: str = "FIRST Inventory"
    data_dir: Path = Path(__file__).resolve().parent.parent / "data"

    @property
    def database_url(self) -> str:
        db_file = self.data_dir / "inventory.db"
        return f"sqlite:///{db_file.as_posix()}"
    label_width_mm: float = 40.0
    label_height_mm: float = 30.0
    cors_origins: list[str] = [
        "http://127.0.0.1:5173",
        "http://localhost:5173",
    ]


settings = Settings()
settings.data_dir.mkdir(parents=True, exist_ok=True)
