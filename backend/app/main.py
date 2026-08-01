from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, JSONResponse
from fastapi.staticfiles import StaticFiles
from starlette.exceptions import HTTPException as StarletteHTTPException

from app.config import settings
from app.database import Base, engine, SessionLocal
from app.models import AppSetting, Operator
from app.routers import (
    dashboard,
    import_export,
    items,
    labels,
    locations,
    operators,
    settings as settings_router,
    transactions,
)


def seed_defaults():
    db = SessionLocal()
    try:
        for key, val in [
            ("label_width_mm", "40"),
            ("label_height_mm", "30"),
        ]:
            if not db.get(AppSetting, key):
                db.add(AppSetting(key=key, value=val))
        if db.query(Operator).count() == 0:
            for name in ["Warehouse", "库管"]:
                db.add(Operator(display_name=name))
        db.commit()
    finally:
        db.close()


@asynccontextmanager
async def lifespan(app: FastAPI):
    Base.metadata.create_all(bind=engine)
    AppSetting.__table__.create(bind=engine, checkfirst=True)
    Operator.__table__.create(bind=engine, checkfirst=True)
    seed_defaults()
    yield


app = FastAPI(title=settings.app_name, lifespan=lifespan)


@app.exception_handler(StarletteHTTPException)
async def http_exception_handler(request: Request, exc: StarletteHTTPException):
    return JSONResponse(status_code=exc.status_code, content={"detail": exc.detail})


@app.exception_handler(Exception)
async def unhandled_exception(request: Request, exc: Exception):
    if isinstance(exc, HTTPException):
        return JSONResponse(status_code=exc.status_code, content={"detail": exc.detail})
    if request.url.path.startswith("/api/"):
        return JSONResponse(
            status_code=500,
            content={
                "detail": str(exc),
                "path": request.url.path,
                "type": type(exc).__name__,
            },
        )
    raise exc


app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins + ["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(items.router)
app.include_router(transactions.router)
app.include_router(operators.router)
app.include_router(locations.router)
app.include_router(settings_router.router)
app.include_router(import_export.router)
app.include_router(labels.router)
app.include_router(dashboard.router)


@app.get("/api/health")
def health():
    return {"status": "ok"}


_FRONTEND_DIST = Path(__file__).resolve().parents[2] / "frontend" / "dist"


def _mount_frontend():
    """Static UI on port 8000. API routes are registered above and take priority."""
    if not (_FRONTEND_DIST / "index.html").is_file():
        return

    assets = _FRONTEND_DIST / "assets"
    if assets.is_dir():
        app.mount("/assets", StaticFiles(directory=assets), name="assets")

    @app.get("/")
    async def spa_index():
        return FileResponse(_FRONTEND_DIST / "index.html")


_mount_frontend()


@app.middleware("http")
async def spa_history_fallback(request: Request, call_next):
    """Vue hash router: only / is required. For history mode paths, serve index.html."""
    path = request.url.path
    if (
        request.method == "GET"
        and not path.startswith("/api")
        and not path.startswith("/assets")
        and path not in ("/", "/docs", "/openapi.json", "/redoc")
        and (_FRONTEND_DIST / "index.html").is_file()
        and "." not in Path(path).name
    ):
        return FileResponse(_FRONTEND_DIST / "index.html")
    return await call_next(request)
