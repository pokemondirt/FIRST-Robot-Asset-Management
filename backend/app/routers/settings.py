from fastapi import APIRouter, Depends
from sqlalchemy import inspect
from sqlalchemy.orm import Session, joinedload

from app.config import settings as app_settings
from app.database import engine, get_db
from app.models import AppSetting, Item
from app.schemas import SettingsRead, SettingsUpdate, StatsResponse

router = APIRouter(prefix="/api/settings", tags=["settings"])


def _ensure_settings_table():
    if not inspect(engine).has_table("app_settings"):
        AppSetting.__table__.create(bind=engine, checkfirst=True)


def _get_setting(db: Session, key: str, default: str) -> str:
    row = db.get(AppSetting, key)
    return row.value if row and row.value is not None else default


def _safe_float(value: str, default: float) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return default


def _set_setting(db: Session, key: str, value: str):
    row = db.get(AppSetting, key)
    if row:
        row.value = value
    else:
        db.add(AppSetting(key=key, value=value))
    db.commit()


@router.get("", response_model=SettingsRead)
def get_settings(db: Session = Depends(get_db)):
    _ensure_settings_table()
    return SettingsRead(
        label_width_mm=_safe_float(
            _get_setting(db, "label_width_mm", str(app_settings.label_width_mm)),
            app_settings.label_width_mm,
        ),
        label_height_mm=_safe_float(
            _get_setting(db, "label_height_mm", str(app_settings.label_height_mm)),
            app_settings.label_height_mm,
        ),
    )


@router.patch("", response_model=SettingsRead)
def update_settings(data: SettingsUpdate, db: Session = Depends(get_db)):
    _ensure_settings_table()
    if data.label_width_mm is not None:
        _set_setting(db, "label_width_mm", str(data.label_width_mm))
    if data.label_height_mm is not None:
        _set_setting(db, "label_height_mm", str(data.label_height_mm))
    return get_settings(db)


@router.get("/stats", response_model=StatsResponse)
def get_stats(db: Session = Depends(get_db)):
    items = (
        db.query(Item)
        .options(joinedload(Item.stock))
        .filter(Item.active == True)  # noqa: E712
        .all()
    )
    low = 0
    total_qty = 0
    for item in items:
        qty = item.stock.quantity if item.stock else 0
        total_qty += qty
        if item.min_stock > 0 and qty < item.min_stock:
            low += 1
    return StatsResponse(
        total_items=len(items),
        low_stock_count=low,
        total_quantity=total_qty,
    )
